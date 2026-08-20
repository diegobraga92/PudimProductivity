package featureflag

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

const featureFlagCacheTTL = 30 * time.Second

func RegisterFeatureFlagRoutes(r chi.Router, repo Repository, auditLogger audit.Logger) *Service {
	service := NewService(repo, featureFlagCacheTTL)
	handler := NewHandler(service, auditLogger)

	r.Route("/api/v1/features", func(r chi.Router) {
		r.Get("/", handler.ListEnabled)
		r.Get("/{name}", handler.GetByName)
		// Toggling flags is admin-only, enforced by httpx.RequireRole.
		// In development the AuthMiddleware trusts X-User-Role; production will
		// validate JWTs.
		r.With(httpx.RequireRole("admin")).Put("/{name}/toggle", handler.Toggle)
	})

	log.Info().Msg("feature flag module routes registered")
	return service
}
