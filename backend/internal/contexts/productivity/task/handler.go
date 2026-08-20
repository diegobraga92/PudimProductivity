package task

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/nlp"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/optional"
)

const defaultCompletionsLookbackDays = 7

type Service interface {
	CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error)
	CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error)
	CreateTaskWithSchedule(ctx context.Context, title string, recurrenceDays []string, listID *string, startTime, endTime, color, scheduledDate *string, alarmMinutes *int, clientID *string) (*Task, error)
	GetTask(ctx context.Context, id string) (*Task, error)
	ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error)
	ListScheduledTasks(ctx context.Context) ([]*Task, error)
	ListTasksByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error)
	UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, error)
	MergeTask(ctx context.Context, id, userID string, clientUpdatedAt time.Time, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, bool, error)
	DeleteTask(ctx context.Context, id string) error
	CompleteTask(ctx context.Context, taskID, dateStr string, completionID *string) (*TaskCompletion, error)
	UncompleteTask(ctx context.Context, taskID, dateStr string) error
	GetTaskCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error)
	GetAllTaskCompletions(ctx context.Context, from, to time.Time) ([]*TaskCompletion, error)
	GetTodayCompletion(ctx context.Context, taskID string) (*TaskCompletion, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// DTO
type TaskResponse struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	RecurrenceDays []string `json:"recurrence_days,omitempty"`
	ListID         *string  `json:"list_id,omitempty"`
	StartTime      *string  `json:"start_time,omitempty"`
	EndTime        *string  `json:"end_time,omitempty"`
	Color          *string  `json:"color,omitempty"`
	ScheduledDate  *string  `json:"scheduled_date,omitempty"`
	AlarmMinutes   *int     `json:"alarm_minutes,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type createTaskRequest struct {
	ID             string   `json:"id,omitempty"`
	Title          string   `json:"title"`
	RecurrenceDays []string `json:"recurrence_days,omitempty"`
	ListID         *string  `json:"list_id,omitempty"`
	StartTime      *string  `json:"start_time,omitempty"`
	EndTime        *string  `json:"end_time,omitempty"`
	Color          *string  `json:"color,omitempty"`
	ScheduledDate  *string  `json:"scheduled_date,omitempty"`
	AlarmMinutes   *int     `json:"alarm_minutes,omitempty"`
}

type updateTaskRequest struct {
	UpdatedAt      time.Time                 `json:"updated_at"`
	Title          *string                   `json:"title"`
	Status         *TaskStatus               `json:"status"`
	RecurrenceDays *[]string                 `json:"recurrence_days"`
	ListID         optional.Optional[string] `json:"list_id"`
	StartTime      optional.Optional[string] `json:"start_time"`
	EndTime        optional.Optional[string] `json:"end_time"`
	Color          optional.Optional[string] `json:"color"`
	ScheduledDate  optional.Optional[string] `json:"scheduled_date"`
	AlarmMinutes   optional.Optional[int]    `json:"alarm_minutes"`
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
		StartTime:      t.StartTime,
		EndTime:        t.EndTime,
		Color:          t.Color,
		ScheduledDate:  t.ScheduledDate,
		AlarmMinutes:   t.AlarmMinutes,
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
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	responses := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = ToTaskResponse(t)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// GET /api/v1/tasks/scheduled — returns all tasks with time-blocking info for the planner
func (h *Handler) ListScheduledTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.service.ListScheduledTasks(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list scheduled tasks")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list scheduled tasks")
		return
	}

	responses := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = ToTaskResponse(t)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// POST /api/v1/tasks
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		httpx.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}

	var clientID *string
	if req.ID != "" {
		clientID = &req.ID
	}

	task, err := h.service.CreateTaskWithSchedule(r.Context(), req.Title, req.RecurrenceDays, req.ListID, req.StartTime, req.EndTime, req.Color, req.ScheduledDate, req.AlarmMinutes, clientID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, ToTaskResponse(task))
}

// GET /api/v1/tasks/{taskId}
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ToTaskResponse(task))
}

// PUT /api/v1/tasks/{taskId}
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.service.UpdateTask(r.Context(), id, req.Title, req.Status, req.RecurrenceDays, req.ListID.Ptr(), req.StartTime.Ptr(), req.EndTime.Ptr(), req.Color.Ptr(), req.ScheduledDate.Ptr(), req.AlarmMinutes.Ptr())
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to update task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ToTaskResponse(task))
}

// PATCH /api/v1/tasks/{taskId}/merge — CRDT merge (Phase 8, ADR 010).
// Body is an updateTaskRequest plus an optional client `updated_at` timestamp.
// Responses:
//
//	200 — this write won; body is the merged task.
//	409 — this write lost (a newer writer exists); body is the winning task.
func (h *Handler) MergeTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := httpx.GetUserID(r.Context())
	task, applied, err := h.service.MergeTask(r.Context(), id, userID, req.UpdatedAt,
		req.Title, req.Status, req.RecurrenceDays, req.ListID.Ptr(),
		req.StartTime.Ptr(), req.EndTime.Ptr(), req.Color.Ptr(), req.ScheduledDate.Ptr(), req.AlarmMinutes.Ptr())
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to merge task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to merge task")
		return
	}

	if !applied {
		// Lost the merge — return the winning state so the client reconciles.
		httpx.WriteJSON(w, http.StatusConflict, ToTaskResponse(task))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ToTaskResponse(task))
}

// DELETE /api/v1/tasks/{taskId}
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	if err := h.service.DeleteTask(r.Context(), id); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to delete task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/tasks/{taskId}/complete
func (h *Handler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	dateStr := r.URL.Query().Get("date")
	var completionID *string
	if idParam := r.URL.Query().Get("id"); idParam != "" {
		completionID = &idParam
	}

	completion, err := h.service.CompleteTask(r.Context(), id, dateStr, completionID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionAlreadyExists) {
			httpx.WriteError(w, http.StatusConflict, "task already completed for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to complete task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to complete task")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toCompletionResponse(completion))
}

// DELETE /api/v1/tasks/{taskId}/complete
func (h *Handler) UncompleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
		return
	}

	dateStr := r.URL.Query().Get("date")

	if err := h.service.UncompleteTask(r.Context(), id, dateStr); err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		if errors.Is(err, ErrCompletionNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "no completion found for this date")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to uncomplete task")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to uncomplete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/tasks/completions
func (h *Handler) GetAllCompletions(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -defaultCompletionsLookbackDays).Truncate(24 * time.Hour)
	to := now.Truncate(24 * time.Hour)

	if fromStr != "" {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
			return
		}
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	}

	completions, err := h.service.GetAllTaskCompletions(r.Context(), from, to)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all task completions")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// GET /api/v1/tasks/{taskId}/completions
func (h *Handler) GetTaskCompletions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "task ID is required")
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
			httpx.WriteError(w, http.StatusBadRequest, "invalid 'from' date format, use YYYY-MM-DD")
			return
		}
	} else {
		// Default to 7 days ago
		from = now.AddDate(0, 0, -defaultCompletionsLookbackDays).Truncate(24 * time.Hour)
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid 'to' date format, use YYYY-MM-DD")
			return
		}
	} else {
		to = now.Truncate(24 * time.Hour)
	}

	completions, err := h.service.GetTaskCompletions(r.Context(), id, from, to)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Error().Err(err).Str("task_id", id).Msg("failed to get task completions")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get task completions")
		return
	}

	responses := make([]taskCompletionResponse, len(completions))
	for i, c := range completions {
		responses[i] = toCompletionResponse(c)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// --- NLP parse endpoint (Phase 7) ---

type ParseTaskRequest struct {
	Input string `json:"input"`
}

type ParseTaskResponse struct {
	Title           string   `json:"title,omitempty"`
	DueDate         *string  `json:"due_date,omitempty"`
	StartTime       *string  `json:"start_time,omitempty"`
	EndTime         *string  `json:"end_time,omitempty"`
	DurationMinutes int      `json:"duration_minutes,omitempty"`
	RecurrenceDays  []string `json:"recurrence_days,omitempty"`
}

// ParseTask parses a natural-language task input and returns the extracted
// fields so the client can pre-fill its creation form.
func (h *Handler) ParseTask(w http.ResponseWriter, r *http.Request) {
	var req ParseTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Input == "" {
		httpx.WriteError(w, http.StatusBadRequest, "input is required")
		return
	}

	parsed, err := nlp.Parse(req.Input, time.Now())
	if errors.Is(err, nlp.ErrNoParse) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "could not parse input — try adding a date, time or duration")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("parse task input failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to parse input")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ParseTaskResponse{
		Title:           parsed.Title,
		DueDate:         parsed.DueDate,
		StartTime:       parsed.StartTime,
		EndTime:         parsed.EndTime,
		DurationMinutes: parsed.DurationMinutes,
		RecurrenceDays:  parsed.RecurrenceDays,
	})
}
