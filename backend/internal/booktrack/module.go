package booktrack

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterBookRoutes wires the Book Tracking module. lookup is optional: pass
// nil to run in degraded mode (by-ISBN lookup returns 502; manual entry works).
func RegisterBookRoutes(r chi.Router, pool *pgxpool.Pool, lookup LookupClient, auditLogger audit.Logger, bus eventbus.Bus) *BookService {
	repo := NewPostgresRepository(pool)
	service := NewBookService(repo, lookup, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/books", func(r chi.Router) {
		// Read-only endpoints — anonymous access allowed.
		r.Get("/", handler.List)
		r.Get("/{bookId}", handler.Get)

		// Mutations require an authenticated user (dev identity headers).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.AddManual)
			r.Post("/by-isbn", handler.AddByISBN)
			r.Put("/{bookId}/status", handler.UpdateStatus)
			r.Delete("/{bookId}", handler.Delete)
		})
	})

	log.Info().Msg("booktrack module routes registered")
	return service
}
