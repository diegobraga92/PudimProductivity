package library

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterLibraryRoutes wires the Library endpoints
func RegisterLibraryRoutes(r chi.Router, repo Repository, auditLogger audit.Logger, bus eventbus.Bus, lookup ScoreLookupProvider, flags *featureflag.Service) *LibraryService {
	service := NewLibraryService(repo, auditLogger, bus)
	handler := NewHandler(service, lookup, flags)

	r.Route("/api/v1/library", func(r chi.Router) {
		// Read-only endpoints — anonymous access allowed.
		r.Get("/", handler.List)
		r.Get("/subtypes", handler.Subtypes)
		r.Get("/score/search", handler.SearchScores)
		r.Post("/score/batch", handler.SearchScoresBatch)
		r.Get("/{itemId}", handler.Get)

		// Mutations require an authenticated user.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.Create)
			r.Post("/import", handler.Import)
			r.Put("/{itemId}", handler.Update)
			r.Delete("/{itemId}", handler.Delete)
		})
	})

	log.Info().Msg("library module routes registered")
	return service
}
