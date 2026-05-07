package task

import (
	"fmt"
	"time"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// ValidTaskStatuses contains all valid task statuses.
var ValidTaskStatuses = map[TaskStatus]bool{
	TaskStatusTodo:       true,
	TaskStatusInProgress: true,
	TaskStatusDone:       true,
}

// TaskPriority represents the priority level of a task.
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

// ValidTaskPriorities contains all valid task priorities.
var ValidTaskPriorities = map[TaskPriority]bool{
	TaskPriorityLow:    true,
	TaskPriorityMedium: true,
	TaskPriorityHigh:   true,
}

// Task is the core domain aggregate for the task bounded context.
type Task struct {
	ID          string
	Title       string
	Description *string
	Status      TaskStatus
	Priority    TaskPriority
	DueDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewTask creates a new Task with the given title and optional fields.
// It returns an error if the title is empty.
func NewTask(id, title string, description *string, priority TaskPriority, dueDate *time.Time) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}
	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}
	if !ValidTaskPriorities[priority] {
		return nil, fmt.Errorf("invalid task priority: %s", priority)
	}

	now := time.Now().UTC()
	return &Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      TaskStatusTodo,
		Priority:    priority,
		DueDate:     dueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update applies the provided updates to the task.
// Only non-nil fields are applied. Returns an error if the resulting state is invalid.
func (t *Task) Update(title *string, description *string, status *TaskStatus, priority *TaskPriority, dueDate **time.Time) error {
	if title != nil {
		if *title == "" {
			return fmt.Errorf("task title cannot be empty")
		}
		t.Title = *title
	}
	if description != nil {
		t.Description = description // allows setting to nil
	}
	if status != nil {
		if !ValidTaskStatuses[*status] {
			return fmt.Errorf("invalid task status: %s", *status)
		}
		t.Status = *status
	}
	if priority != nil {
		if !ValidTaskPriorities[*priority] {
			return fmt.Errorf("invalid task priority: %s", *priority)
		}
		t.Priority = *priority
	}
	if dueDate != nil {
		t.DueDate = *dueDate // allows setting to nil
	}

	t.UpdatedAt = time.Now().UTC()
	return nil
}

// IsOverdue returns true if the task has a due date in the past and is not done.
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil || t.Status == TaskStatusDone {
		return false
	}
	return time.Now().After(*t.DueDate)
}
