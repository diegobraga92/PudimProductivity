package scoring

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterScoreProviderRoutes wires the admin score-provider settings module.
func RegisterScoreProviderRoutes(r chi.Router, repo Repository, auditLogger audit.Logger, flags *featureflag.Service, envCfg config.ScoreProviderConfig) (*Service, *Manager) {
	manager := NewManager(nil)
	service := NewService(repo, flags, manager, auditLogger, envCfg)
	handler := NewHandler(service)

	r.Route("/api/v1/admin/score-providers", func(r chi.Router) {
		// Read + write both require admin.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin"))
			r.Get("/", handler.Get)
			r.Put("/", handler.Update)
		})
	})

	log.Info().Msg("score provider settings module routes registered")
	return service, manager
}
