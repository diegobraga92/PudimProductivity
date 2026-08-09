package featureflag

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

const featureFlagCacheTTL = 30 * time.Second

func RegisterFeatureFlagRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger) *Service {
	repo := NewPostgresRepository(pool)
	service := NewService(repo, featureFlagCacheTTL)
	handler := NewHandler(service, auditLogger)

	r.Route("/api/v1/features", func(r chi.Router) {
		r.Get("/", handler.ListEnabled)
		r.Get("/{name}", handler.GetByName)
		// Toggling flags is admin-only, enforced by shared.RequireRole.
		// In development the AuthMiddleware trusts X-User-Role; production will
		// validate JWTs.
		r.With(shared.RequireRole("admin")).Put("/{name}/toggle", handler.Toggle)
	})

	log.Info().Msg("feature flag module routes registered")
	return service
}
