package tasklist

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

type Handler struct {
	service *TaskListService
}

func NewHandler(service *TaskListService) *Handler {
	return &Handler{service: service}
}

// DTO
type taskListResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type createTaskListRequest struct {
	Name string `json:"name"`
}

type updateTaskListRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func toTaskListResponse(l *TaskList) taskListResponse {
	return taskListResponse{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// GET /api/v1/task-lists
func (h *Handler) ListTaskLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.service.ListTaskLists(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list task lists")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list task lists")
		return
	}

	responses := make([]taskListResponse, len(lists))
	for i, l := range lists {
		responses[i] = toTaskListResponse(l)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}

// POST /api/v1/task-lists
func (h *Handler) CreateTaskList(w http.ResponseWriter, r *http.Request) {
	var req createTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		shared.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := h.service.CreateTaskList(r.Context(), req.Name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create task list")
		shared.WriteError(w, http.StatusInternalServerError, "failed to create task list")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, toTaskListResponse(list))
}

// GET /api/v1/task-lists/{listId}
func (h *Handler) GetTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	list, err := h.service.GetTaskList(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to get task list")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get task list")
		return
	}

	shared.WriteJSON(w, http.StatusOK, toTaskListResponse(list))
}

// PUT /api/v1/task-lists/{listId}
func (h *Handler) UpdateTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	var req updateTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	list, err := h.service.UpdateTaskList(r.Context(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to update task list")
		shared.WriteError(w, http.StatusInternalServerError, "failed to update task list")
		return
	}

	shared.WriteJSON(w, http.StatusOK, toTaskListResponse(list))
}

// DELETE /api/v1/task-lists/{listId}
func (h *Handler) DeleteTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	if err := h.service.DeleteTaskList(r.Context(), id); err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to delete task list")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete task list")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/{listId}/tasks
// Returns a handler that lists tasks for a list. Uses task service to fetch tasks by list ID
func (h *Handler) ListTasksByListID(taskService *task.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "listId")
		if id == "" {
			shared.WriteError(w, http.StatusBadRequest, "list ID is required")
			return
		}

		typeFilter := r.URL.Query().Get("type")

		tasks, err := taskService.ListTasksByListID(r.Context(), id, typeFilter)
		if err != nil {
			log.Error().Err(err).Str("list_id", id).Msg("failed to list tasks by list")
			shared.WriteError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		responses := make([]task.TaskResponse, len(tasks))
		for i, t := range tasks {
			responses[i] = task.ToTaskResponse(t)
		}

		shared.WriteJSON(w, http.StatusOK, responses)
	}
}
