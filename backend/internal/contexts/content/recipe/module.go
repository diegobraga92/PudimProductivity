package recipe

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterRecipeRoutes wires the Recipes module. uploads is optional.
func RegisterRecipeRoutes(r chi.Router, repo Repository, auditLogger audit.Logger, bus eventbus.Bus, uploads media.Generator) *RecipeService {
	service := NewService(repo, auditLogger, bus, uploads)
	handler := NewHandler(service)

	r.Route("/api/v1/recipes", func(r chi.Router) {
		// Read-only endpoints.
		r.Get("/", handler.ListRecipes)
		r.Get("/{recipeId}", handler.GetRecipe)

		// Mutations require an authenticated user.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.CreateRecipe)
			r.Put("/{recipeId}", handler.UpdateRecipe)
			r.Delete("/{recipeId}", handler.DeleteRecipe)
			r.Post("/{recipeId}/upload-url", handler.GenerateUploadURL)
		})
	})

	log.Info().Msg("recipe module routes registered")
	return service
}
