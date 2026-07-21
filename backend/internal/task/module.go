package task

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
)

// Returns the TaskService so other modules (e.g., tasklist) can use it.
func RegisterTaskRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger) *TaskService {
	repo := NewPostgresTaskRepository(pool)
	service := NewTaskService(repo, auditLogger)
	handler := NewHandler(service)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Get("/", handler.ListTasks)
		r.Post("/", handler.CreateTask)
		// Scheduled tasks endpoint for planner view — must be before /{taskId} to
		// prevent chi from matching the literal "scheduled" as a taskId.
		r.Get("/scheduled", handler.ListScheduledTasks)
		// Batch completions endpoint — must be registered before /{taskId}
		r.Get("/completions", handler.GetAllCompletions)
		r.Get("/{taskId}", handler.GetTask)
		r.Put("/{taskId}", handler.UpdateTask)
		r.Delete("/{taskId}", handler.DeleteTask)
		r.Post("/{taskId}/complete", handler.CompleteTask)
		r.Delete("/{taskId}/complete", handler.UncompleteTask)
		r.Get("/{taskId}/completions", handler.GetTaskCompletions)
	})

	log.Info().Msg("task module routes registered")
	return service
}
