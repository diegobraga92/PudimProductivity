package tasklist

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

func RegisterTaskListRoutes(r chi.Router, repo TaskListRepository, bus eventbus.Bus, taskService *task.TaskService) {
	service := NewTaskListService(repo, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/task-lists", func(r chi.Router) {
		// Read-only endpoints.
		r.Get("/", handler.ListTaskLists)
		r.Get("/{listId}", handler.GetTaskList)
		r.Get("/{listId}/tasks", handler.ListTasksByListID(taskService))
		r.Get("/{listId}/members", handler.ListMembers)

		// Mutating endpoints require an authenticated user.
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTaskList)
			r.Put("/{listId}", handler.UpdateTaskList)
			r.Delete("/{listId}", handler.DeleteTaskList)
			r.Post("/{listId}/share", handler.ShareTaskList)
			r.Delete("/{listId}/share/{userId}", handler.UnshareTaskList)
		})
	})

	log.Info().Msg("task list module routes registered")
}
