package task

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterTaskRoutes mounts all task HTTP routes on the given router.
// It wires up the repository, service, and handler.
func RegisterTaskRoutes(r chi.Router, pool *pgxpool.Pool, bus shared.EventBus) {
	repo := NewPostgresTaskRepository(pool)
	service := NewTaskService(repo, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Get("/", handler.ListTasks)
		r.Post("/", handler.CreateTask)
		r.Get("/{taskId}", handler.GetTask)
		r.Put("/{taskId}", handler.UpdateTask)
		r.Delete("/{taskId}", handler.DeleteTask)
	})

	log.Info().Msg("task module routes registered")
}
