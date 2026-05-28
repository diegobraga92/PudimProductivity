package tasklist

import (
	"context"
	"errors"
)

var ErrTaskListNotFound = errors.New("task list not found")

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
