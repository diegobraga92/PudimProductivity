package main

import (
	"bufio"
	"context"
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
	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/observability"
	"github.com/diegobraga92/pudimproductivity/backend/internal/pomodoro"
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

	// Setup database
	dbCtx, dbCancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer dbCancel()

	pool, err := db.ConnectPool(dbCtx, cfg.Database)
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
	metrics := shared.NewMetrics()
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

	// Event bus + real-time sync hub (Phase 2). The hub subscribes to the bus
	// and fans events out to connected WebSocket clients.
	bus := eventbus.NewInMemoryBus()
	syncHub := sync.NewHub(bus, sync.Config{ReplayBufferSize: 1000})
	if err := syncHub.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to start sync hub")
	}
	sync.RegisterSyncRoutes(r, syncHub)

	var auditService *audit.Service
	if pool != nil {
		auditRepo := audit.NewPostgresRepository(pool)
		auditService = audit.NewService(auditRepo, 1024)
	}

	// Setup routes
	var taskService *task.TaskService
	if pool != nil {
		taskService = task.RegisterTaskRoutes(r, pool, auditService, bus)
	}

	if pool != nil {
		tasklist.RegisterTaskListRoutes(r, pool, taskService)
	}

	if pool != nil {
		featureflag.RegisterFeatureFlagRoutes(r, pool)
	}

	pomodoro.RegisterPomodoroRoutes(r, nil)

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
	if err := bus.Close(); err != nil {
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
