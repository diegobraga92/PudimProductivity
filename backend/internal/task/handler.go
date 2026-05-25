package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// optionalString distinguishes a JSON field that was absent from one that was
// explicitly null or set to a value.
//
//   - set=false → field was absent; do not change the stored value.
//   - set=true, val=nil → field was explicitly null; clear the stored value.
//   - set=true, val=&s → field was a string; set the stored value to s.
type optionalString struct {
	set bool
	val *string
}

func (o *optionalString) UnmarshalJSON(data []byte) error {
	o.set = true
	if string(data) == "null" {
		o.val = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	o.val = &s
	return nil
}

// ptr returns a **string suitable for Task.Update:
// nil outer pointer if the field was absent, otherwise a pointer to val.
func (o *optionalString) ptr() **string {
	if !o.set {
		return nil
	}
	return &o.val
}

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
	Title          *string        `json:"title"`
	Status         *TaskStatus    `json:"status"`
	RecurrenceDays *[]string      `json:"recurrence_days"`
	ListID         optionalString `json:"list_id"`
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
		if errors.Is(err, ErrTaskNotFound) {
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

	task, err := h.service.UpdateTask(r.Context(), id, req.Title, req.Status, req.RecurrenceDays, req.ListID.ptr())
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
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
		if errors.Is(err, ErrTaskNotFound) {
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

	dateStr := r.URL.Query().Get("date")

	completion, err := h.service.CompleteTask(r.Context(), id, dateStr)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionAlreadyExists) {
			writeError(w, http.StatusConflict, "task already completed for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to complete task")
		writeError(w, http.StatusInternalServerError, "failed to complete task")
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

	dateStr := r.URL.Query().Get("date")

	if err := h.service.UncompleteTask(r.Context(), id, dateStr); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionNotFound) {
			writeError(w, http.StatusNotFound, "no completion found for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to uncomplete task")
		writeError(w, http.StatusInternalServerError, "failed to uncomplete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAllCompletions handles GET /api/v1/tasks/completions
// Returns all completions across every habit task within an optional date range.
// This is a batch alternative that avoids N+1 per-task requests from the client.
func (h *Handler) GetAllCompletions(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
	to := now.Truncate(24 * time.Hour)

	if fromStr != "" {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
			return
		}
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	}

	completions, err := h.service.GetAllTaskCompletions(r.Context(), from, to)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all task completions")
		writeError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	writeJSON(w, http.StatusOK, responses)
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
		if errors.Is(err, ErrTaskNotFound) {
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
