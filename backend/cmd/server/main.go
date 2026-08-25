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
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/notification"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/rabbitmq"
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

	// OpenTelemetry tracing (Phase 6): every HTTP request + event-bus dispatch
	// gets a W3C trace ID, exported to stdout and (optionally) an OTLP collector.
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

	// Prometheus metrics registry — created before the DB pool so every query
	// can be traced (see db.ConnectPoolWithMetrics).
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

	// OpenTelemetry tracing middleware — outermost, so spans are active across
	// the whole chain (metrics, logging, handlers, WebSocket upgrades).
	r.Use(observability.TracingMiddleware)

	// Prometheus metrics: record request metrics on the main router. The scrape
	// endpoint is served by the internal :9090 server (see below) so it is not
	// exposed on the public port.
	r.Use(metrics.MetricsMiddleware)
	r.Use(middleware.RequestID)
	// Note: middleware.RealIP is intentionally omitted — it is deprecated
	// (GHSA-3fxj-6jh8-hvhx) because it blindly trusts X-Forwarded-For and other
	// headers, letting clients spoof their IP. The request logger uses
	// r.RemoteAddr, and production sits behind nginx which sets those headers.
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	// CORS for cross-origin clients (e.g. the Electron desktop app served from
	// the app://bundle origin). Must run before AuthMiddleware so preflight
	// OPTIONS requests are answered without identity headers. No-op when
	// CORS_ALLOWED_ORIGINS is empty (same-origin web deployment).
	r.Use(httpx.CorsMiddleware(cfg.Server.CORSAllowedOrigins))
	r.Use(httpx.AuthMiddleware)
	r.Use(middleware.Timeout(cfg.Server.RequestTimeout))

	r.Get("/api/v1/health", healthHandler(pool, cfg.Version))
	r.Post("/api/v1/errors", clientErrorHandler)

	// Event bus + real-time sync hub (Phase 2). The hub subscribes to the bus
	// and fans events out to connected WebSocket clients.
	inMemoryBus := eventbus.NewInMemoryBus()
	syncHub := sync.NewHub(inMemoryBus, sync.Config{ReplayBufferSize: 1000})
	if err := syncHub.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to start sync hub")
	}
	sync.RegisterSyncRoutes(r, syncHub)

	// Phase 8: collaboration — membership resolver for event scoping + presence,
	// and the presence snapshot endpoint. When the DB pool is unavailable the
	// hub degrades to broadcast (legacy behavior).
	if pool != nil {
		syncHub.SetMembershipResolver(postgres.NewMembershipRepository(pool))
		membership.RegisterCollabRoutes(r, syncHub)
	}

	// Phase 3: RabbitMQ becomes the durable backbone for async consumers. If
	// RabbitMQ is unavailable we degrade gracefully: events still fan out to
	// WebSocket clients via the in-memory bus, but the notifications worker
	// stays off until the broker comes back.
	composite := eventbus.NewCompositeBus(inMemoryBus)
	var rabbitBus *rabbitmq.Bus
	var notifWorker *notification.Worker
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://pudim:" + os.Getenv("RABBITMQ_PASS") + "@rabbitmq:5672/"
	}
	if rb, err := rabbitmq.New(context.Background(), rabbitmq.Config{URL: rabbitURL}); err != nil {
		log.Warn().Err(err).Msg("rabbitmq unavailable — notifications worker disabled")
	} else {
		rabbitBus = rb
		composite = eventbus.NewCompositeBus(inMemoryBus, rb)

		emails := []notification.EmailDeliverer{
			notification.NewEmailSender(notification.EmailConfig{
				SMTPHost: os.Getenv("SMTP_HOST"),
				SMTPPort: os.Getenv("SMTP_PORT"),
				From:     os.Getenv("SMTP_FROM"),
			}),
		}

		var pushes []notification.PushDeliverer
		if fcm, err := notification.NewFCMSender(context.Background(), notification.FCMConfig{}); err != nil {
			log.Warn().Err(err).Msg("fcm sender disabled")
			pushes = append(pushes, notification.NoopSender{})
		} else {
			pushes = append(pushes, fcm)
		}

		var notifRepo notification.Repo = notification.NewMemoryRepo()
		if pool != nil {
			notifRepo = notification.NewPostgresRepo(pool)
		}

		notifWorker = notification.NewWorker(rb, emails, pushes, notifRepo, notification.Recipients{
			Email:     os.Getenv("NOTIFY_EMAIL"),
			PushToken: os.Getenv("FCM_DEVICE_TOKEN"),
		})
		go func() {
			if err := notifWorker.Run(context.Background()); err != nil {
				log.Error().Err(err).Msg("notification worker stopped with error")
			}
		}()
	}

	// Redis: optional shared pub/sub fabric for cross-instance sync fan-out plus
	// a read-through cache for task reads. When Redis is unavailable the sync
	// hub keeps working single-instance and caching is disabled (degraded
	// mode), mirroring the RabbitMQ behavior above.
	var redisBus *eventbus.RedisBus
	if cfg.RedisURL != "" {
		rb, err := eventbus.NewRedisBus(context.Background(), eventbus.RedisConfig{URL: cfg.RedisURL})
		if err != nil {
			log.Warn().Err(err).Msg("redis unavailable — cross-instance sync and task cache disabled")
		} else {
			redisBus = rb
			// Relay events from other instances into the local bus so the sync
			// hub (subscribed to inMemoryBus) broadcasts them to its clients.
			// The Redis bus filters self-origin messages, so there is no echo
			// loop and no double delivery for locally produced events.
			if _, err := rb.Subscribe(context.Background(), func(ctx context.Context, e eventbus.Event) error {
				return inMemoryBus.Publish(ctx, e.Type, e.Payload)
			}); err != nil {
				log.Warn().Err(err).Msg("redis subscribe failed — cross-instance sync disabled")
				redisBus = nil
			} else {
				children := []eventbus.Bus{inMemoryBus, rb}
				if rabbitBus != nil {
					children = append(children, rabbitBus)
				}
				composite = eventbus.NewCompositeBus(children...)
			}
		}
	}

	var auditService *audit.Service
	if pool != nil {
		auditRepo := postgres.NewAuditRepository(pool)
		auditService = audit.NewService(auditRepo, 1024)
	}

	// Setup routes
	// Task read-through cache (optional, Redis-backed). Every task mutation
	// bumps the shared cache version, so stale entries are never served across
	// instances either.
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

	// Phase 9c: offline-first sync — GET /api/v1/sync?since=... returns the
	// incremental changes (active + soft-deleted rows) for mobile Room DBs.
	persistence.RegisterSyncStoreRoutes(r, postgres.NewSyncRepository(pool))

	// Backup & Restore — full snapshot of the non-sensitive data as JSON for
	// disaster recovery (export = download, import = replace backed-up tables).
	if pool != nil {
		backup.RegisterBackupRoutes(r, postgres.NewBackupRepository(pool), cfg.Version)
	}

	// Phase 5a: Recipes. Media uploads are optional. Storage backend selection
	// in priority order: S3_MEDIA_BUCKET (real S3 or any S3-compatible store)
	// → local server disk (MEDIA_STORAGE=local or MEDIA_LOCAL_DIR set). With
	// neither set the module runs in degraded mode and upload-URL endpoints
	// return 503.
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

	// Soundscape ambient sound library (rain, fire, noise loops, …). The default
	// loops ship inside the image (backend/sounds → /app/sounds-default) and are
	// copied into SOUNDS_DIR on startup; existing files are never overwritten,
	// so a sound can be overridden on disk or via the sounds data volume without
	// rebuilding the image. In dev, both dirs default to ./sounds, where seeding
	// is a harmless no-op and the repo files are served directly.
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

	// Library: media tracking (movies, series, books, games) with a done flag,
	// release year, optional notes and an optional score lookup. The provider
	// configuration lives in the database and is editable at runtime through the
	// admin UI (GET/PUT /api/v1/admin/score-providers). The environment variables
	// (SCORE_PROVIDER_MOVIE/SERIES/GAME/BOOK, OMDB_API_KEY / RAWG_API_KEY) act
	// only as a one-time bootstrap until settings are saved in the UI. When
	// nothing is configured the feature runs in degraded mode (score search
	// returns 503), per ADR 007. Replaces the Phase 5 booktrack module.
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

	// Start the internal metrics server on a separate, non-public port (:9090).
	// Prometheus scrapes metrics from here; the public router never exposes /metrics.
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
	if notifWorker != nil {
		notifWorker.Close()
	}
	if rabbitBus != nil {
		if err := rabbitBus.Close(); err != nil {
			log.Warn().Err(err).Msg("rabbitmq bus close error")
		}
	}
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

// clientErrorRequest is the payload sent by web/mobile error reporters
// (POST /api/v1/errors).
type clientErrorRequest struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	Context string `json:"context,omitempty"`
}

// clientErrorHandler ingests client-side errors (web window.onerror +
// unhandledrejection, mobile uncaught exceptions) and logs them with the
// request's trace context. Returns 202 — the client needs no feedback.
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
