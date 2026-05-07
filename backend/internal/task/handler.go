package task

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Handler contains HTTP handlers for task operations.
type Handler struct {
	service *TaskService
}

// NewHandler creates a new Handler.
func NewHandler(service *TaskService) *Handler {
	return &Handler{service: service}
}

// --- DTOs ---

type taskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type createTaskRequest struct {
	Title       string      `json:"title"`
	Description *string     `json:"description"`
	Priority    TaskPriority `json:"priority"`
	DueDate     *string     `json:"due_date"`
}

type updateTaskRequest struct {
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	Status      *TaskStatus   `json:"status"`
	Priority    *TaskPriority `json:"priority"`
	DueDate     *string       `json:"due_date"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Helpers ---

func toTaskResponse(t *Task) taskResponse {
	resp := taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Priority:    string(t.Priority),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
	if t.DueDate != nil {
		formatted := t.DueDate.Format(time.RFC3339)
		resp.DueDate = &formatted
	}
	return resp
}

func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
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

// ListTasks handles GET /api/v1/tasks
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	priorityFilter := r.URL.Query().Get("priority")

	tasks, err := h.service.ListTasks(r.Context(), statusFilter, priorityFilter)
	if err != nil {
		log.Error().Err(err).Msg("failed to list tasks")
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	responses := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = toTaskResponse(t)
	}

	writeJSON(w, http.StatusOK, responses)
}

// CreateTask handles POST /api/v1/tasks
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = TaskPriorityMedium
	}

	dueDate := parseTimePtr(req.DueDate)

	task, err := h.service.CreateTask(r.Context(), req.Title, req.Description, priority, dueDate)
	if err != nil {
		log.Error().Err(err).Msg("failed to create task")
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	writeJSON(w, http.StatusCreated, toTaskResponse(task))
}

// GetTask handles GET /api/v1/tasks/{taskId}
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task")
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, toTaskResponse(task))
}

// UpdateTask handles PUT /api/v1/tasks/{taskId}
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert string dueDate to time.Time pointer
	var dueDate **time.Time
	if req.DueDate != nil {
		parsed := parseTimePtr(req.DueDate)
		dueDate = &parsed
	}

	task, err := h.service.UpdateTask(r.Context(), id, req.Title, req.Description, req.Status, req.Priority, dueDate)
	if err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	writeJSON(w, http.StatusOK, toTaskResponse(task))
}

// DeleteTask handles DELETE /api/v1/tasks/{taskId}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	if err := h.service.DeleteTask(r.Context(), id); err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to delete task")
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
