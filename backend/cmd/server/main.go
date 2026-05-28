package main

import (
	"context"
	"fmt"
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

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/features"
	"github.com/diegobraga92/pudimproductivity/backend/internal/pomodoro"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
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
		Logger()

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

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
		migrateCtx, migrateCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.WriteTimeout)*time.Second)
		defer migrateCancel()

		if err := db.RunMigrations(migrateCtx, pool); err != nil {
			log.Fatal().Err(err).Msg("failed to run database migrations")
		}
	}

	// Setup feature flag service
	var featureStore *features.CachedFeatureStore
	if pool != nil {
		pgFeatureStore := features.NewPostgresFeatureStore(pool)
		featureStore = features.NewCachedFeatureStore(pgFeatureStore, 30*time.Second)
	}

	// Setup router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/api/v1/health", healthHandler(pool, cfg.Version))

	// Setup routes
	var taskService *task.TaskService
	if pool != nil {
		taskService = task.RegisterTaskRoutes(r, pool)
	}

	if pool != nil {
		tasklist.RegisterTaskListRoutes(r, pool, taskService)
	}

	if featureStore != nil {
		features.RegisterFeatureRoutes(r, featureStore)
	}

	pomodoro.RegisterPomodoroRoutes(r, nil)

	// Setup server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
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

	// Wait for shutdown signal
	sig := <-shutdownCh
	log.Info().Str("signal", sig.String()).Msg("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
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
