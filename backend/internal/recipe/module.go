package recipe

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterRecipeRoutes wires the Recipes module. uploads is optional: pass nil
// to run in degraded mode (upload-URL endpoints return 503).
func RegisterRecipeRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger, bus eventbus.Bus, uploads media.Generator) *RecipeService {
	repo := NewPostgresRepository(pool)
	service := NewService(repo, auditLogger, bus, uploads)
	handler := NewHandler(service)

	r.Route("/api/v1/recipes", func(r chi.Router) {
		// Read-only endpoints — anonymous access allowed.
		r.Get("/", handler.ListRecipes)
		r.Get("/{recipeId}", handler.GetRecipe)

		// Mutations require an authenticated user (dev identity headers).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.CreateRecipe)
			r.Put("/{recipeId}", handler.UpdateRecipe)
			r.Delete("/{recipeId}", handler.DeleteRecipe)
			r.Post("/{recipeId}/upload-url", handler.GenerateUploadURL)
		})
	})

	log.Info().Msg("recipe module routes registered")
	return service
}
