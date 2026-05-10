package tasklist

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

// RegisterTaskListRoutes mounts all task list HTTP routes on the given router.
// It accepts the task service to support listing tasks within a list.
func RegisterTaskListRoutes(r chi.Router, pool *pgxpool.Pool, taskService *task.TaskService) {
	repo := NewPostgresTaskListRepository(pool)
	service := NewTaskListService(repo)
	handler := NewHandler(service)

	r.Route("/api/v1/task-lists", func(r chi.Router) {
		r.Get("/", handler.ListTaskLists)
		r.Post("/", handler.CreateTaskList)
		r.Get("/{listId}", handler.GetTaskList)
		r.Put("/{listId}", handler.UpdateTaskList)
		r.Delete("/{listId}", handler.DeleteTaskList)
		r.Get("/{listId}/tasks", handler.ListTasksByListID(taskService))
	})

	log.Info().Msg("task list module routes registered")
}
