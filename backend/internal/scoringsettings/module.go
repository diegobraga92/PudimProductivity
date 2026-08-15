package scoringsettings

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/library/scoring"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterScoreProviderRoutes wires the admin score-provider settings module.
// It returns the settings service and the runtime lookup Manager so the caller
// can ApplyConfig at startup and inject the Manager into the library module.
func RegisterScoreProviderRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger, flags *featureflag.Service, envCfg shared.ScoreProviderConfig) (*Service, *scoring.Manager) {
	repo := NewPostgresRepository(pool)
	manager := scoring.NewManager(nil)
	service := NewService(repo, flags, manager, auditLogger, envCfg)
	handler := NewHandler(service)

	r.Route("/api/v1/admin/score-providers", func(r chi.Router) {
		// Read + write both require admin (dev identity header X-User-Role: admin).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin"))
			r.Get("/", handler.Get)
			r.Put("/", handler.Update)
		})
	})

	log.Info().Msg("score provider settings module routes registered")
	return service, manager
}
