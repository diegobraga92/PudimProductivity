// Package syncstore (Phase 9c) serves the incremental offline-sync endpoint:
// GET /api/v1/sync?since=RFC3339 returns every task, completion, task list and
// share changed (created/updated/soft-deleted) after `since`, plus a new
// timestamp for the client to store. Offline clients (mobile Room DB) use this
// to converge with the server after being disconnected.
package syncstore

import "time"

// Bundle is the full incremental payload. Deleted IDs are tombstones produced
// by the soft-delete migration (019); share keys are "list_id:shared_with"
// composites (task_list_shares has a composite PK, no single id column).
type Bundle struct {
	Timestamp            string          `json:"timestamp"`
	Tasks                []TaskDTO       `json:"tasks"`
	DeletedTaskIDs       []string        `json:"deleted_task_ids"`
	Completions          []CompletionDTO `json:"completions"`
	DeletedCompletionIDs []string        `json:"deleted_completion_ids"`
	TaskLists            []TaskListDTO   `json:"task_lists"`
	DeletedTaskListIDs   []string        `json:"deleted_task_list_ids"`
	Shares               []ShareDTO      `json:"shares"`
	DeletedShareKeys     []string        `json:"deleted_share_keys"`
}

// TaskDTO mirrors the task API response shape (see task.TaskResponse).
type TaskDTO struct {
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

// CompletionDTO mirrors the completions API response shape.
type CompletionDTO struct {
	ID            string `json:"id"`
	TaskID        string `json:"task_id"`
	CompletedDate string `json:"completed_date"` // "2006-01-02"
	CreatedAt     string `json:"created_at"`
}

// TaskListDTO mirrors the task-list API response shape.
type TaskListDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ShareDTO mirrors the members API response shape.
type ShareDTO struct {
	ListID     string    `json:"list_id"`
	SharedWith string    `json:"shared_with"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}
