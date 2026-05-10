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
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	RecurrenceDays []string `json:"recurrence_days,omitempty"`
	ListID         *string  `json:"list_id,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type createTaskRequest struct {
	Title          string   `json:"title"`
	RecurrenceDays []string `json:"recurrence_days,omitempty"`
	ListID         *string  `json:"list_id,omitempty"`
}

type updateTaskRequest struct {
	Title          *string     `json:"title"`
	Status         *TaskStatus `json:"status"`
	RecurrenceDays *[]string   `json:"recurrence_days"`
	ListID         *string     `json:"list_id,omitempty"`
}

type taskCompletionResponse struct {
	ID            string `json:"id"`
	TaskID        string `json:"task_id"`
	CompletedDate string `json:"completed_date"`
	CreatedAt     string `json:"created_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Helpers ---

func toTaskResponse(t *Task) taskResponse {
	return taskResponse{
		ID:             t.ID,
		Title:          t.Title,
		Status:         string(t.Status),
		RecurrenceDays: t.RecurrenceDays,
		ListID:         t.ListID,
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
	}
}

func toCompletionResponse(c *TaskCompletion) taskCompletionResponse {
	return taskCompletionResponse{
		ID:            c.ID,
		TaskID:        c.TaskID,
		CompletedDate: c.CompletedDate.Format("2006-01-02"),
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
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

// ListTasks handles GET /api/v1/tasks
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	typeFilter := r.URL.Query().Get("type")

	tasks, err := h.service.ListTasks(r.Context(), statusFilter, typeFilter)
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

	task, err := h.service.CreateTaskWithList(r.Context(), req.Title, req.RecurrenceDays, req.ListID)
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

	task, err := h.service.UpdateTask(r.Context(), id, req.Title, req.Status, req.RecurrenceDays)
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

// CompleteTask handles POST /api/v1/tasks/{taskId}/complete
func (h *Handler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	completion, err := h.service.CompleteTask(r.Context(), id)
	if err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to complete task")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toCompletionResponse(completion))
}

// UncompleteTask handles DELETE /api/v1/tasks/{taskId}/complete
func (h *Handler) UncompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	if err := h.service.UncompleteTask(r.Context(), id); err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to uncomplete task")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTaskCompletions handles GET /api/v1/tasks/{taskId}/completions
func (h *Handler) GetTaskCompletions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	// Parse optional date range query params
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now().UTC()
	var from, to time.Time

	if fromStr != "" {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
			return
		}
	} else {
		// Default to 7 days ago
		from = now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	} else {
		to = now.Truncate(24 * time.Hour)
	}

	completions, err := h.service.GetTaskCompletions(r.Context(), id, from, to)
	if err != nil {
		if err == ErrTaskNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task completions")
		writeError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	writeJSON(w, http.StatusOK, responses)
}
