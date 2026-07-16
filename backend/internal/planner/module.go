package planner

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func RegisterPlannerRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewPostgresPlannerRepository(pool)
	service := NewPlannerService(repo)
	handler := NewHandler(service)

	r.Route("/api/v1/planner", func(r chi.Router) {
		r.Get("/", handler.ListPlannerEntries)
		r.Post("/", handler.CreatePlannerEntry)
		r.Get("/{entryId}", handler.GetPlannerEntry)
		r.Put("/{entryId}", handler.UpdatePlannerEntry)
		r.Delete("/{entryId}", handler.DeletePlannerEntry)
	})

	log.Info().Msg("planner module routes registered")
}
