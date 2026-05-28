package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service *TaskService
}

func NewHandler(service *TaskService) *Handler {
	return &Handler{service: service}
}

// DTO
type TaskResponse struct {
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
	Title          *string                 `json:"title"`
	Status         *TaskStatus             `json:"status"`
	RecurrenceDays *[]string               `json:"recurrence_days"`
	ListID         shared.Optional[string] `json:"list_id"`
}

type taskCompletionResponse struct {
	ID            string `json:"id"`
	TaskID        string `json:"task_id"`
	CompletedDate string `json:"completed_date"`
	CreatedAt     string `json:"created_at"`
}

func ToTaskResponse(t *Task) TaskResponse {
	return TaskResponse{
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

// GET /api/v1/tasks
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	typeFilter := r.URL.Query().Get("type")

	tasks, err := h.service.ListTasks(r.Context(), statusFilter, typeFilter)
	if err != nil {
		log.Error().Err(err).Msg("failed to list tasks")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	responses := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = ToTaskResponse(t)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}

// POST /api/v1/tasks
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		shared.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := h.service.CreateTaskWithList(r.Context(), req.Title, req.RecurrenceDays, req.ListID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, ToTaskResponse(task))
}

// GET /api/v1/tasks/{taskId}
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	shared.WriteJSON(w, http.StatusOK, ToTaskResponse(task))
}

// PUT /api/v1/tasks/{taskId}
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.service.UpdateTask(r.Context(), id, req.Title, req.Status, req.RecurrenceDays, req.ListID.Ptr())
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	shared.WriteJSON(w, http.StatusOK, ToTaskResponse(task))
}

// DELETE /api/v1/tasks/{taskId}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	if err := h.service.DeleteTask(r.Context(), id); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to delete task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/tasks/{taskId}/complete
func (h *Handler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	dateStr := r.URL.Query().Get("date")

	completion, err := h.service.CompleteTask(r.Context(), id, dateStr)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionAlreadyExists) {
			shared.WriteError(w, http.StatusConflict, "task already completed for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to complete task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to complete task")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, toCompletionResponse(completion))
}

// DELETE /api/v1/tasks/{taskId}/complete
func (h *Handler) UncompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	dateStr := r.URL.Query().Get("date")

	if err := h.service.UncompleteTask(r.Context(), id, dateStr); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionNotFound) {
			shared.WriteError(w, http.StatusNotFound, "no completion found for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to uncomplete task")
		shared.WriteError(w, http.StatusInternalServerError, "failed to uncomplete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/tasks/completions
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
			shared.WriteError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
			return
		}
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	}

	completions, err := h.service.GetAllTaskCompletions(r.Context(), from, to)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all task completions")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}

// GET /api/v1/tasks/{taskId}/completions
func (h *Handler) GetTaskCompletions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		shared.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now().UTC()
	var from, to time.Time

	if fromStr != "" {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
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
			shared.WriteError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	} else {
		to = now.Truncate(24 * time.Hour)
	}

	completions, err := h.service.GetTaskCompletions(r.Context(), id, from, to)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			shared.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task completions")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}
