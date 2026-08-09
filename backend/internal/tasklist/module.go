package tasklist

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

func RegisterTaskListRoutes(r chi.Router, pool *pgxpool.Pool, taskService *task.TaskService) {
	repo := NewPostgresTaskListRepository(pool)
	service := NewTaskListService(repo)
	handler := NewHandler(service)

	r.Route("/api/v1/task-lists", func(r chi.Router) {
		// Read-only endpoints.
		r.Get("/", handler.ListTaskLists)
		r.Get("/{listId}", handler.GetTaskList)
		r.Get("/{listId}/tasks", handler.ListTasksByListID(taskService))

		// Mutating endpoints require an authenticated user (see task module).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTaskList)
			r.Put("/{listId}", handler.UpdateTaskList)
			r.Delete("/{listId}", handler.DeleteTaskList)
		})
	})

	log.Info().Msg("task list module routes registered")
}
