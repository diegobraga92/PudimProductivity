package tasklist

import (
	"fmt"
	"time"
)

// TaskList represents a named grouping of tasks (e.g. "Shopping List", "Project Ideas").
type TaskList struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTaskList creates a new TaskList with the given name.
func NewTaskList(id, name string) (*TaskList, error) {
	if id == "" {
		return nil, fmt.Errorf("task list id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("task list name cannot be empty")
	}

	now := time.Now().UTC()
	return &TaskList{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update applies the provided updates to the task list.
func (l *TaskList) Update(name, description *string) error {
	if name != nil {
		if *name == "" {
			return fmt.Errorf("task list name cannot be empty")
		}
		l.Name = *name
	}
	if description != nil {
		l.Description = *description
	}
	l.UpdatedAt = time.Now().UTC()
	return nil
}
