package featureflag

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func RegisterFeatureFlagRoutes(r chi.Router, pool *pgxpool.Pool) *Service {
	repo := NewPostgresRepository(pool)
	service := NewService(repo, 30*time.Second) // 30s cache TTL
	handler := NewHandler(service)

	r.Route("/api/v1/features", func(r chi.Router) {
		r.Get("/", handler.ListEnabled)
		r.Get("/{name}", handler.GetByName)
		r.Put("/{name}/toggle", handler.Toggle) // TODO: add admin RBAC middleware
	})

	log.Info().Msg("feature flag module routes registered")
	return service
}