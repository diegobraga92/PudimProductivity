package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/backup"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/membership"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/sync"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/sync/persistence"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/recipe"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/scoring"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/sounds"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/pomodoro"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/storage"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/cache"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/observability"
)

func main() {
	// Setup logging and config
	cfg := config.LoadConfig()
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Caller().
		Logger().
		Hook(observability.TraceLogHook{})

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	// OpenTelemetry tracing.
	tp, err := observability.InitTracing(context.Background(), observability.Config{
		ServiceName:  "pudim-backend",
		Version:      cfg.Version,
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Stdout:       true,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to initialize OpenTelemetry tracing — continuing without spans")
	} else {
		defer func() { _ = tp.Shutdown(context.Background()) }()
	}

	log.Info().Str("version", cfg.Version).Msg("starting PudimProductivity backend")

	// Prometheus metrics registry.
	metrics := observability.NewMetrics()

	// Setup database
	dbCtx, dbCancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer dbCancel()

	pool, err := postgres.ConnectPoolWithMetrics(dbCtx, cfg.Database, metrics)
	if err != nil {
		log.Warn().Err(err).Msg("database connection failed — running in degraded mode")
		pool = nil
	}

	if pool != nil {
		migrateCtx, migrateCancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
		defer migrateCancel()

		if err := postgres.RunMigrations(migrateCtx, pool); err != nil {
			log.Fatal().Err(err).Msg("failed to run database migrations")
		}
	}

	// Setup router
	r := chi.NewRouter()

	// OpenTelemetry tracing middleware.
	r.Use(observability.TracingMiddleware)

	// Prometheus metrics.
	r.Use(metrics.MetricsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	// CORS for cross-origin clients (e.g. the Electron desktop app served from
	// the app://bundle origin).
	r.Use(httpx.CorsMiddleware(cfg.Server.CORSAllowedOrigins))
	r.Use(httpx.AuthMiddleware)
	r.Use(middleware.Timeout(cfg.Server.RequestTimeout))

	r.Get("/api/v1/health", healthHandler(pool, cfg.Version))
	r.Post("/api/v1/errors", clientErrorHandler)

	// Event bus + real-time sync hub.
	inMemoryBus := eventbus.NewInMemoryBus()
	syncHub := sync.NewHub(inMemoryBus, sync.Config{ReplayBufferSize: 1000})
	if err := syncHub.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to start sync hub")
	}
	sync.RegisterSyncRoutes(r, syncHub)

	// Membership resolver for event scoping + presence.
	if pool != nil {
		syncHub.SetMembershipResolver(postgres.NewMembershipRepository(pool))
		membership.RegisterCollabRoutes(r, syncHub)
	}

	// Event bus fan-out.
	composite := eventbus.NewCompositeBus(inMemoryBus)

	// Redis for optional shared pub/sub fabric for cross-instance sync fan-out.
	var redisBus *eventbus.RedisBus
	if cfg.RedisURL != "" {
		rb, err := eventbus.NewRedisBus(context.Background(), eventbus.RedisConfig{URL: cfg.RedisURL})
		if err != nil {
			log.Warn().Err(err).Msg("redis unavailable — cross-instance sync and task cache disabled")
		} else {
			redisBus = rb
			if _, err := rb.Subscribe(context.Background(), func(ctx context.Context, e eventbus.Event) error {
				return inMemoryBus.Publish(ctx, e.Type, e.Payload)
			}); err != nil {
				log.Warn().Err(err).Msg("redis subscribe failed — cross-instance sync disabled")
				redisBus = nil
			} else {
				composite = eventbus.NewCompositeBus(inMemoryBus, rb)
			}
		}
	}

	var auditService *audit.Service
	if pool != nil {
		auditRepo := postgres.NewAuditRepository(pool)
		auditService = audit.NewService(auditRepo, 1024)
	}

	// Setup routes.
	var taskCache *cache.Cache
	if pool != nil && redisBus != nil {
		tc, err := cache.New(context.Background(), cfg.RedisURL, cfg.RedisCacheTTL)
		if err != nil {
			log.Warn().Err(err).Msg("redis task cache unavailable — task reads hit the database")
		} else {
			taskCache = tc
		}
	}

	var taskService *task.TaskService
	if pool != nil {
		taskService = task.RegisterTaskRoutes(r, postgres.NewTaskRepository(pool), auditService, composite, taskCache)
	}

	if pool != nil {
		tasklist.RegisterTaskListRoutes(r, postgres.NewTaskListRepository(pool), composite, taskService)
	}

	var flagService *featureflag.Service
	if pool != nil {
		flagService = featureflag.RegisterFeatureFlagRoutes(r, postgres.NewFeatureFlagRepository(pool), auditService)
	}

	pomodoro.RegisterPomodoroRoutes(r, nil, auditService, composite)
	persistence.RegisterSyncStoreRoutes(r, postgres.NewSyncRepository(pool))

	if pool != nil {
		backup.RegisterBackupRoutes(r, postgres.NewBackupRepository(pool), cfg.Version)
	}

	var uploads media.Generator
	if bucket := os.Getenv("S3_MEDIA_BUCKET"); bucket != "" {
		region := os.Getenv("S3_MEDIA_REGION")
		if region == "" {
			region = "us-east-1"
		}
		s3Uploader, err := storage.NewS3Uploader(context.Background(), bucket, region)
		if err != nil {
			log.Warn().Err(err).Msg("media uploads disabled — S3 not configured")
		} else {
			uploads = s3Uploader
		}
	} else if os.Getenv("MEDIA_STORAGE") == "local" || os.Getenv("MEDIA_LOCAL_DIR") != "" {
		dir := os.Getenv("MEDIA_LOCAL_DIR")
		if dir == "" {
			dir = "./data/media"
		}
		localUploader, err := storage.NewFilesystemUploader(dir, os.Getenv("MEDIA_PUBLIC_BASE_URL"))
		if err != nil {
			log.Warn().Err(err).Msg("media uploads disabled — local storage not configured")
		} else {
			uploads = localUploader
			media.RegisterMediaRoutes(r, dir)
		}
	} else {
		log.Info().Msg("no media storage configured — recipe media uploads disabled (degraded mode)")
	}

	soundsDir := os.Getenv("SOUNDS_DIR")
	if soundsDir == "" {
		soundsDir = "./sounds"
	}
	bundledSoundsDir := os.Getenv("SOUNDS_BUNDLED_DIR")
	if bundledSoundsDir == "" {
		bundledSoundsDir = "./sounds"
	}
	if err := sounds.SeedBundledDefaults(bundledSoundsDir, soundsDir); err != nil {
		log.Warn().Err(err).Msg("soundscape defaults seeding failed — serving directory may be incomplete")
	}
	sounds.RegisterSoundsRoutes(r, soundsDir, sounds.DefaultCatalog)

	// Phase 5a: Recipes — depends on the media uploader (optional) for images.
	if pool != nil {
		recipe.RegisterRecipeRoutes(r, postgres.NewRecipeRepository(pool), auditService, composite, uploads)
	}

	if pool != nil {
		settingsService, scoreManager := scoring.RegisterScoreProviderRoutes(r, postgres.NewScoringRepository(pool), auditService, flagService, config.LoadScoreProviderConfig())
		if err := settingsService.ApplyConfig(context.Background()); err != nil {
			log.Warn().Err(err).Msg("library score lookup disabled — invalid provider config")
		}
		library.RegisterLibraryRoutes(r, postgres.NewLibraryRepository(pool), auditService, composite, scoreManager, flagService)
	}

	// Setup server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownCh)

	// Start server async (background goroutine)
	go func() {
		log.Info().Str("addr", addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Start the internal metrics server on a separate, non-public port.
	internalMetricsServer := observability.SetupInternalMetricsServer(metrics)
	go func() {
		log.Info().Str("addr", internalMetricsServer.Addr).Msg("internal metrics server listening")
		if err := internalMetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("internal metrics server error")
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdownCh
	log.Info().Str("signal", sig.String()).Msg("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	if err := internalMetricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("internal metrics server shutdown error")
	}

	syncHub.Close()
	if redisBus != nil {
		if err := redisBus.Close(); err != nil {
			log.Warn().Err(err).Msg("redis event bus close error")
		}
	}
	if taskCache != nil {
		if err := taskCache.Close(); err != nil {
			log.Warn().Err(err).Msg("task cache close error")
		}
	}
	if err := inMemoryBus.Close(); err != nil {
		log.Warn().Err(err).Msg("event bus close error")
	}

	if pool != nil {
		pool.Close()
		log.Info().Msg("database pool closed")
	}

	log.Info().Msg("server stopped")
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			log.Info().
				Ctx(r.Context()).
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Dur("duration", time.Since(start)).
				Str("remote", r.RemoteAddr).
				Msg("request completed")
		}()

		next.ServeHTTP(wrapped, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack lets the WebSocket upgrade pass through the logger wrapper. Without
// it, http.Hijacker support would be lost and upgrades would fail with 501.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hj.Hijack()
}

// Flush forwards flushes to the underlying writer, if supported.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	DB      string `json:"db"`
}

func healthHandler(pool *pgxpool.Pool, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "connected"
		overallStatus := "ok"

		if pool == nil {
			dbStatus = "disconnected"
			overallStatus = "degraded"
		} else {
			pingCtx, pingCancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer pingCancel()
			if err := pool.Ping(pingCtx); err != nil {
				dbStatus = "disconnected"
				overallStatus = "degraded"
			}
		}

		statusCode := http.StatusOK
		if overallStatus == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}

		httpx.WriteJSON(w, statusCode, HealthResponse{
			Status:  overallStatus,
			Version: version,
			DB:      dbStatus,
		})
	}
}

// clientErrorRequest is the payload sent by web/mobile error reporters.
type clientErrorRequest struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	Context string `json:"context,omitempty"`
}

// clientErrorHandler ingests client-side errors.
func clientErrorHandler(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()

	var req clientErrorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid error payload")
		return
	}

	source := r.Header.Get("X-Error-Source")
	if source == "" {
		source = "unknown"
	}

	ev := log.Error().
		Ctx(r.Context()).
		Str("source", source).
		Str("error_context", req.Context)
	if req.Stack != "" {
		ev = ev.Str("stack", req.Stack)
	}
	ev.Msg("client error reported: " + req.Message)

	w.WriteHeader(http.StatusAccepted)
}
