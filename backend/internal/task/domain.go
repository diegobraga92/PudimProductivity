package task

import (
	"fmt"
	"time"
)

// TaskStatus represents whether a task is still to do or already done.
type TaskStatus string

const (
	TaskStatusTodo TaskStatus = "todo"
	TaskStatusDone TaskStatus = "done"
)

// ValidTaskStatuses contains all valid task statuses.
var ValidTaskStatuses = map[TaskStatus]bool{
	TaskStatusTodo: true,
	TaskStatusDone: true,
}

// ValidRecurrenceDays contains all valid day-of-week abbreviations.
var ValidRecurrenceDays = map[string]bool{
	"mon": true,
	"tue": true,
	"wed": true,
	"thu": true,
	"fri": true,
	"sat": true,
	"sun": true,
}

// Task is the core domain aggregate for the task bounded context.
// If RecurrenceDays is non-nil, the task is a habit that repeats on those days.
// If RecurrenceDays is nil, the task is a one-off task with a simple todo/done status.
// ListID is optional and links the task to a named task list.
type Task struct {
	ID             string
	Title          string
	Status         TaskStatus
	RecurrenceDays []string // nil = one-off task, non-nil = habit
	ListID         *string  // nil = not part of a list, non-nil = belongs to a task list
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewTask creates a new Task with the given title and optional recurrence days.
// It returns an error if the title is empty or if recurrence days are invalid.
func NewTask(id, title string, recurrenceDays []string) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}

	if recurrenceDays != nil {
		if len(recurrenceDays) == 0 {
			return nil, fmt.Errorf("recurrence days cannot be empty; use nil for one-off tasks")
		}
		for _, d := range recurrenceDays {
			if !ValidRecurrenceDays[d] {
				return nil, fmt.Errorf("invalid recurrence day: %s", d)
			}
		}
	}

	now := time.Now().UTC()
	return &Task{
		ID:             id,
		Title:          title,
		Status:         TaskStatusTodo,
		RecurrenceDays: recurrenceDays,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// IsHabit returns true if this task has recurrence days (i.e., it's a habit).
func (t *Task) IsHabit() bool {
	return t.RecurrenceDays != nil
}

// Update applies the provided updates to the task.
// Only non-nil fields are applied. Returns an error if the resulting state is invalid.
func (t *Task) Update(title *string, status *TaskStatus, recurrenceDays *[]string) error {
	if title != nil {
		if *title == "" {
			return fmt.Errorf("task title cannot be empty")
		}
		t.Title = *title
	}
	if status != nil {
		if !ValidTaskStatuses[*status] {
			return fmt.Errorf("invalid task status: %s", *status)
		}
		t.Status = *status
	}
	if recurrenceDays != nil {
		if len(*recurrenceDays) == 0 {
			t.RecurrenceDays = nil
		} else {
			for _, d := range *recurrenceDays {
				if !ValidRecurrenceDays[d] {
					return fmt.Errorf("invalid recurrence day: %s", d)
				}
			}
			t.RecurrenceDays = *recurrenceDays
		}
	}

	t.UpdatedAt = time.Now().UTC()
	return nil
}

// TaskCompletion represents a single day's completion of a habit task.
type TaskCompletion struct {
	ID            string
	TaskID        string
	CompletedDate time.Time
	CreatedAt     time.Time
}

// NewTaskCompletion creates a new TaskCompletion.
func NewTaskCompletion(id, taskID string, completedDate time.Time) (*TaskCompletion, error) {
	if id == "" {
		return nil, fmt.Errorf("completion id cannot be empty")
	}
	if taskID == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}

	return &TaskCompletion{
		ID:            id,
		TaskID:        taskID,
		CompletedDate: completedDate,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
