package tasklist

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

func RegisterTaskListRoutes(r chi.Router, pool *pgxpool.Pool, bus eventbus.Bus, taskService *task.TaskService) {
	repo := NewPostgresTaskListRepository(pool)
	service := NewTaskListService(repo, bus)
	handler := NewHandler(service)

	r.Route("/api/v1/task-lists", func(r chi.Router) {
		// Read-only endpoints.
		r.Get("/", handler.ListTaskLists)
		r.Get("/{listId}", handler.GetTaskList)
		r.Get("/{listId}/tasks", handler.ListTasksByListID(taskService))
		// Phase 8: collaboration — members are readable by any member.
		r.Get("/{listId}/members", handler.ListMembers)

		// Mutating endpoints require an authenticated user (see task module).
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Post("/", handler.CreateTaskList)
			r.Put("/{listId}", handler.UpdateTaskList)
			r.Delete("/{listId}", handler.DeleteTaskList)
			// Phase 8: sharing. Note: /{listId}/share and /{listId}/members
			// must be registered before any /{listId} catch-alls (chi does
			// not match across path segments, so this is safe).
			r.Post("/{listId}/share", handler.ShareTaskList)
			r.Delete("/{listId}/share/{userId}", handler.UnshareTaskList)
		})
	})

	log.Info().Msg("task list module routes registered")
}
