package task

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Returns the TaskService so other modules (e.g., tasklist) can use it.
func RegisterTaskRoutes(r chi.Router, pool *pgxpool.Pool, auditLogger audit.Logger, bus eventbus.Bus) *TaskService {
	repo := NewPostgresTaskRepository(pool)
	service := NewTaskService(repo, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		// Read-only endpoints — available to anonymous and authenticated users.
		r.Get("/", handler.ListTasks)
		// Scheduled tasks endpoint for planner view — must be before /{taskId} to
		// prevent chi from matching the literal "scheduled" as a taskId.
		r.Get("/scheduled", handler.ListScheduledTasks)
		// Batch completions endpoint — must be registered before /{taskId}
		r.Get("/completions", handler.GetAllCompletions)
		r.Get("/{taskId}", handler.GetTask)
		r.Get("/{taskId}/completions", handler.GetTaskCompletions)

		// Mutating endpoints require an authenticated user. In development the
		// AuthMiddleware trusts X-User-ID / X-User-Role headers; production will
		// validate JWTs. Anonymous callers get HTTP 403.
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTask)
			r.Put("/{taskId}", handler.UpdateTask)
			r.Delete("/{taskId}", handler.DeleteTask)
			r.Post("/{taskId}/complete", handler.CompleteTask)
			r.Delete("/{taskId}/complete", handler.UncompleteTask)
		})
	})

	log.Info().Msg("task module routes registered")
	return service
}
