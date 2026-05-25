package tasklist

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

// Handler contains HTTP handlers for task list operations.
type Handler struct {
	service *TaskListService
}

// NewHandler creates a new Handler.
func NewHandler(service *TaskListService) *Handler {
	return &Handler{service: service}
}

// --- DTOs ---

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

type errorResponse struct {
	Error string `json:"error"`
}

// --- Helpers ---

func toTaskListResponse(l *TaskList) taskListResponse {
	return taskListResponse{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// --- Handlers ---

// ListTaskLists handles GET /api/v1/task-lists
func (h *Handler) ListTaskLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.service.ListTaskLists(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list task lists")
		writeError(w, http.StatusInternalServerError, "failed to list task lists")
		return
	}

	responses := make([]taskListResponse, len(lists))
	for i, l := range lists {
		responses[i] = toTaskListResponse(l)
	}

	writeJSON(w, http.StatusOK, responses)
}

// CreateTaskList handles POST /api/v1/task-lists
func (h *Handler) CreateTaskList(w http.ResponseWriter, r *http.Request) {
	var req createTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := h.service.CreateTaskList(r.Context(), req.Name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create task list")
		writeError(w, http.StatusInternalServerError, "failed to create task list")
		return
	}

	writeJSON(w, http.StatusCreated, toTaskListResponse(list))
}

// GetTaskList handles GET /api/v1/task-lists/{listId}
func (h *Handler) GetTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	list, err := h.service.GetTaskList(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			writeError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to get task list")
		writeError(w, http.StatusInternalServerError, "failed to get task list")
		return
	}

	writeJSON(w, http.StatusOK, toTaskListResponse(list))
}

// UpdateTaskList handles PUT /api/v1/task-lists/{listId}
func (h *Handler) UpdateTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	var req updateTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	list, err := h.service.UpdateTaskList(r.Context(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			writeError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to update task list")
		writeError(w, http.StatusInternalServerError, "failed to update task list")
		return
	}

	writeJSON(w, http.StatusOK, toTaskListResponse(list))
}

// DeleteTaskList handles DELETE /api/v1/task-lists/{listId}
func (h *Handler) DeleteTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	if err := h.service.DeleteTaskList(r.Context(), id); err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			writeError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to delete task list")
		writeError(w, http.StatusInternalServerError, "failed to delete task list")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListTasksByListID returns a handler that lists tasks for a given task list.
// It uses the task service to fetch tasks by list ID.
func (h *Handler) ListTasksByListID(taskService *task.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "listId")
		if id == "" {
			writeError(w, http.StatusBadRequest, "list ID is required")
			return
		}

		typeFilter := r.URL.Query().Get("type")

		tasks, err := taskService.ListTasksByListID(r.Context(), id, typeFilter)
		if err != nil {
			log.Error().Err(err).Str("list_id", id).Msg("failed to list tasks by list")
			writeError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		// Reuse the task package's response type
		type taskItem struct {
			ID             string   `json:"id"`
			Title          string   `json:"title"`
			Status         string   `json:"status"`
			RecurrenceDays []string `json:"recurrence_days,omitempty"`
			ListID         *string  `json:"list_id,omitempty"`
			CreatedAt      string   `json:"created_at"`
			UpdatedAt      string   `json:"updated_at"`
		}

		responses := make([]taskItem, len(tasks))
		for i, t := range tasks {
			responses[i] = taskItem{
				ID:             t.ID,
				Title:          t.Title,
				Status:         string(t.Status),
				RecurrenceDays: t.RecurrenceDays,
				ListID:         t.ListID,
				CreatedAt:      t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:      t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}

		writeJSON(w, http.StatusOK, responses)
	}
}
