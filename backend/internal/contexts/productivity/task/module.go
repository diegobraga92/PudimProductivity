package task

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterTaskRoutes register the task endpoints.
func RegisterTaskRoutes(r chi.Router, repo TaskRepository, auditLogger audit.Logger, bus eventbus.Bus, cache ...TaskCache) *TaskService {
	service := NewTaskService(repo, auditLogger, bus, cache...)
	handler := NewHandler(service)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Get("/", handler.ListTasks)
		r.Post("/parse", handler.ParseTask)
		r.Get("/scheduled", handler.ListScheduledTasks)
		r.Get("/completions", handler.GetAllCompletions)
		r.Get("/{taskId}", handler.GetTask)
		r.Get("/{taskId}/completions", handler.GetTaskCompletions)

		// Mutating endpoints require an authenticated user.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTask)
			r.Put("/{taskId}", handler.UpdateTask)
			r.Patch("/{taskId}/merge", handler.MergeTask)
			r.Delete("/{taskId}", handler.DeleteTask)
			r.Post("/{taskId}/complete", handler.CompleteTask)
			r.Delete("/{taskId}/complete", handler.UncompleteTask)
		})
	})

	log.Info().Msg("task module routes registered")
	return service
}
