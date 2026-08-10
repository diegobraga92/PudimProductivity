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

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/booktrack"
	"github.com/diegobraga92/pudimproductivity/backend/internal/booktrack/googlebooks"
	"github.com/diegobraga92/pudimproductivity/backend/internal/collab"
	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/mealplan"
	"github.com/diegobraga92/pudimproductivity/backend/internal/notification"
	"github.com/diegobraga92/pudimproductivity/backend/internal/observability"
	"github.com/diegobraga92/pudimproductivity/backend/internal/pomodoro"
	"github.com/diegobraga92/pudimproductivity/backend/internal/rabbitmq"
	"github.com/diegobraga92/pudimproductivity/backend/internal/recipe"
	"github.com/diegobraga92/pudimproductivity/backend/internal/scheduler"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/diegobraga92/pudimproductivity/backend/internal/sync"
	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/tasklist"
)

func main() {
	// Setup logging and config
	cfg := shared.LoadConfig()
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
	metrics := shared.NewMetrics()

	// Setup database
	dbCtx, dbCancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer dbCancel()

	pool, err := db.ConnectPoolWithMetrics(dbCtx, cfg.Database, metrics)
	if err != nil {
		log.Warn().Err(err).Msg("database connection failed — running in degraded mode")
		pool = nil
	}

	if pool != nil {
		migrateCtx, migrateCancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
		defer migrateCancel()

		if err := db.RunMigrations(migrateCtx, pool); err != nil {
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
	r.Use(shared.AuthMiddleware)
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
		syncHub.SetMembershipResolver(collab.NewPostgresMembershipResolver(pool))
		collab.RegisterCollabRoutes(r, syncHub)
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

	var auditService *audit.Service
	if pool != nil {
		auditRepo := audit.NewPostgresRepository(pool)
		auditService = audit.NewService(auditRepo, 1024)
	}

	// Setup routes
	var taskService *task.TaskService
	if pool != nil {
		taskService = task.RegisterTaskRoutes(r, pool, auditService, composite)
	}

	// Phase 7: auto-scheduler — derives a profile from task data and suggests a
	// daily plan. Requires the task service.
	if taskService != nil {
		scheduler.RegisterSchedulerRoutes(r, taskService)
	}

	if pool != nil {
		tasklist.RegisterTaskListRoutes(r, pool, composite, taskService)
	}

	if pool != nil {
		featureflag.RegisterFeatureFlagRoutes(r, pool, auditService)
	}

	pomodoro.RegisterPomodoroRoutes(r, nil, auditService)

	// Phase 5a: Recipes. Media uploads are optional — when S3_MEDIA_BUCKET is
	// unset the module runs in degraded mode and upload-URL endpoints return 503.
	var uploads media.Generator
	if bucket := os.Getenv("S3_MEDIA_BUCKET"); bucket != "" {
		region := os.Getenv("S3_MEDIA_REGION")
		if region == "" {
			region = "us-east-1"
		}
		s3Uploader, err := media.NewS3Uploader(context.Background(), bucket, region)
		if err != nil {
			log.Warn().Err(err).Msg("media uploads disabled — S3 not configured")
		} else {
			uploads = s3Uploader
		}
	} else {
		log.Info().Msg("S3_MEDIA_BUCKET unset — recipe media uploads disabled")
	}
	var recipeService *recipe.RecipeService
	if pool != nil {
		recipeService = recipe.RegisterRecipeRoutes(r, pool, auditService, composite, uploads)
	}

	// Phase 5: Book tracking. The Google Books adapter is always wired; with no
	// GOOGLE_BOOKS_API_KEY it uses the anonymous (rate-limited) endpoint. A nil
	// lookup would degrade by-ISBN entry to 502 while keeping manual entry.
	if pool != nil {
		gb := googlebooks.NewClient(googlebooks.Config{
			APIKey: os.Getenv("GOOGLE_BOOKS_API_KEY"),
		})
		booktrack.RegisterBookRoutes(r, pool, gb, auditService, composite)
	}

	// Phase 5: Meal planning — depends on the recipes module for shopping-list
	// generation (recipeService satisfies mealplan.RecipeReader).
	if pool != nil {
		mealplan.RegisterMealPlanRoutes(r, pool, recipeService, auditService, composite)
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
	internalMetricsServer := shared.SetupInternalMetricsServer(metrics)
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

		shared.WriteJSON(w, statusCode, HealthResponse{
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
		shared.WriteError(w, http.StatusBadRequest, "invalid error payload")
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
