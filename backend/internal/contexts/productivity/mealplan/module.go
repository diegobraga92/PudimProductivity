package mealplan

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterMealPlanRoutes wires the Meal Planner module.
func RegisterMealPlanRoutes(r chi.Router, repo Repository, auditLogger audit.Logger, bus eventbus.Bus) *MealPlanService {
	service := NewService(repo, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/meal-plan", func(r chi.Router) {
		// Read-only endpoints.
		r.Get("/", handler.ListItems)

		// Mutations require an authenticated user.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.CreateItem)
			r.Put("/{itemId}", handler.UpdateItem)
			r.Delete("/{itemId}", handler.DeleteItem)
		})
	})

	log.Info().Msg("meal plan module routes registered")
	return service
}
