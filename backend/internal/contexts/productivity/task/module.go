package task

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// Returns the TaskService so other modules (e.g., tasklist) can use it.
func RegisterTaskRoutes(r chi.Router, repo TaskRepository, auditLogger audit.Logger, bus eventbus.Bus) *TaskService {
	service := NewTaskService(repo, auditLogger, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		// Read-only endpoints — available to anonymous and authenticated users.
		r.Get("/", handler.ListTasks)
		// NLP parse endpoint (Phase 7) — pure transform, no user data.
		r.Post("/parse", handler.ParseTask)
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
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTask)
			r.Put("/{taskId}", handler.UpdateTask)
			// Phase 8: CRDT merge endpoint (LWW register, ADR 010).
			r.Patch("/{taskId}/merge", handler.MergeTask)
			r.Delete("/{taskId}", handler.DeleteTask)
			r.Post("/{taskId}/complete", handler.CompleteTask)
			r.Delete("/{taskId}/complete", handler.UncompleteTask)
		})
	})

	log.Info().Msg("task module routes registered")
	return service
}
