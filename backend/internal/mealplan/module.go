package mealplan

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterMealPlanRoutes wires the Meal Planning module. recipes must be the
// recipe service (satisfies RecipeReader); pass nil to disable shopping-list
// generation.
func RegisterMealPlanRoutes(r chi.Router, pool *pgxpool.Pool, recipes RecipeReader, auditLogger audit.Logger, bus eventbus.Bus) *MealPlanService {
	repo := NewPostgresRepository(pool)
	service := NewMealPlanService(repo, recipes, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/mealplans", func(r chi.Router) {
		r.Get("/", handler.List)
		r.Get("/{planId}", handler.Get)

		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.Create)
			r.Put("/{planId}", handler.Update)
			r.Delete("/{planId}", handler.Delete)
			r.Put("/{planId}/slots/{slotId}", handler.AssignSlot)
			r.Post("/{planId}/shopping-list", handler.GenerateShoppingList)
			r.Get("/{planId}/shopping-list", handler.GetShoppingList)
			r.Put("/{planId}/shopping-list/{itemId}", handler.ToggleShoppingItem)
			r.Post("/{planId}/publish", handler.Publish)
		})
	})

	log.Info().Msg("mealplan module routes registered")
	return service
}
