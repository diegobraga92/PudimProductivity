package task

import (
	"context"
	"time"
)

// TaskRepository defines the interface for task persistence.
// This is the "port" in ports & adapters terminology.
type TaskRepository interface {
	// Create persists a new task.
	Create(ctx context.Context, task *Task) error

	// GetByID retrieves a task by its ID.
	// Returns ErrTaskNotFound if the task does not exist.
	GetByID(ctx context.Context, id string) (*Task, error)

	// List returns all tasks, optionally filtered by status.
	// Passing an empty string for statusFilter means "no filter".
	List(ctx context.Context, statusFilter string) ([]*Task, error)

	// Update persists changes to an existing task.
	// Returns ErrTaskNotFound if the task does not exist.
	Update(ctx context.Context, task *Task) error

	// Delete removes a task by its ID.
	// Returns ErrTaskNotFound if the task does not exist.
	Delete(ctx context.Context, id string) error

	// CreateCompletion records a habit completion for a specific date.
	// Returns an error if a completion already exists for that task+date.
	CreateCompletion(ctx context.Context, completion *TaskCompletion) error

	// DeleteCompletion removes a habit completion for a specific task+date.
	// Returns ErrTaskNotFound if no completion exists.
	DeleteCompletion(ctx context.Context, taskID string, date time.Time) error

	// GetCompletion retrieves a single completion for a task on a specific date.
	// Returns nil (no error) if no completion exists.
	GetCompletion(ctx context.Context, taskID string, date time.Time) (*TaskCompletion, error)

	// ListCompletions returns all completions for a task within a date range (inclusive).
	ListCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error)
}
