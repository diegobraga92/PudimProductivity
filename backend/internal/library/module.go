package library

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterLibraryRoutes wires the Library module (media tracking).
func RegisterLibraryRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger, bus eventbus.Bus) *LibraryService {
	repo := NewPostgresRepository(pool)
	service := NewLibraryService(repo, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/library", func(r chi.Router) {
		// Read-only endpoints — anonymous access allowed.
		r.Get("/", handler.List)
		r.Get("/{itemId}", handler.Get)

		// Mutations require an authenticated user (dev identity headers).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.Create)
			r.Post("/import", handler.Import)
			r.Put("/{itemId}", handler.Update)
			r.Delete("/{itemId}", handler.Delete)
		})
	})

	log.Info().Msg("library module routes registered")
	return service
}
