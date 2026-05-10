package tasklist

import "context"

// ErrTaskListNotFound is returned when a task list is not found.
var ErrTaskListNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string {
	return "task list not found"
}

// TaskListRepository defines the interface for task list persistence.
type TaskListRepository interface {
	// Create persists a new task list.
	Create(ctx context.Context, list *TaskList) error

	// GetByID retrieves a task list by its ID.
	// Returns ErrTaskListNotFound if the list does not exist.
	GetByID(ctx context.Context, id string) (*TaskList, error)

	// List returns all task lists.
	List(ctx context.Context) ([]*TaskList, error)

	// Update persists changes to an existing task list.
	// Returns ErrTaskListNotFound if the list does not exist.
	Update(ctx context.Context, list *TaskList) error

	// Delete removes a task list by its ID.
	// Returns ErrTaskListNotFound if the list does not exist.
	Delete(ctx context.Context, id string) error
}
