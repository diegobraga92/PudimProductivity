package task

import "context"

// TaskRepository defines the interface for task persistence.
// This is the "port" in ports & adapters terminology.
type TaskRepository interface {
	// Create persists a new task.
	Create(ctx context.Context, task *Task) error

	// GetByID retrieves a task by its ID.
	// Returns ErrTaskNotFound if the task does not exist.
	GetByID(ctx context.Context, id string) (*Task, error)

	// List returns all tasks, optionally filtered by status and/or priority.
	// Passing empty strings for filters means "no filter".
	List(ctx context.Context, statusFilter, priorityFilter string) ([]*Task, error)

	// Update persists changes to an existing task.
	// Returns ErrTaskNotFound if the task does not exist.
	Update(ctx context.Context, task *Task) error

	// Delete removes a task by its ID.
	// Returns ErrTaskNotFound if the task does not exist.
	Delete(ctx context.Context, id string) error
}
