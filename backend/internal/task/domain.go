package task

import (
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskStatusTodo TaskStatus = "todo"
	TaskStatusDone TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusDone:
		return true
	default:
		return false
	}
}

var validRecurrenceDays = map[string]struct{}{
	"mon": {},
	"tue": {},
	"wed": {},
	"thu": {},
	"fri": {},
	"sat": {},
	"sun": {},
}

type Task struct {
	ID             string
	Title          string
	Status         TaskStatus
	RecurrenceDays []string // nil = one-off task, non-nil = habit
	ListID         *string  // nil = not part of a list, non-nil = belongs to a task list
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewTask(id, title string, recurrenceDays []string) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}

	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}

	if err := validateRecurrenceDays(recurrenceDays); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Task{
		ID:             id,
		Title:          title,
		Status:         TaskStatusTodo,
		RecurrenceDays: cloneStrings(recurrenceDays),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (t *Task) IsHabit() bool {
	return t.RecurrenceDays != nil
}

func (t *Task) Update(
	title *string,
	status *TaskStatus,
	recurrenceDays *[]string,
	listID **string,
) error {
	if title != nil {
		if *title == "" {
			return fmt.Errorf("task title cannot be empty")
		}

		t.Title = *title
	}

	if status != nil {
		if !status.Valid() {
			return fmt.Errorf("invalid task status: %s", *status)
		}

		t.Status = *status
	}

	if recurrenceDays != nil {
		if err := validateRecurrenceDays(*recurrenceDays); err != nil {
			return err
		}

		t.RecurrenceDays = cloneStrings(*recurrenceDays)
	}

	if listID != nil {
		t.ListID = *listID
	}

	t.UpdatedAt = time.Now().UTC()

	return nil
}

type TaskCompletion struct {
	ID            string
	TaskID        string
	CompletedDate time.Time
	CreatedAt     time.Time
}

func NewTaskCompletion(
	id, taskID string,
	completedDate time.Time,
) (*TaskCompletion, error) {
	if id == "" {
		return nil, fmt.Errorf("completion id cannot be empty")
	}

	if taskID == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}

	if completedDate.IsZero() {
		return nil, fmt.Errorf("completed date cannot be zero")
	}

	return &TaskCompletion{
		ID:            id,
		TaskID:        taskID,
		CompletedDate: completedDate.UTC(),
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func validateRecurrenceDays(days []string) error {
	if days == nil {
		return nil
	}

	if len(days) == 0 {
		return fmt.Errorf("recurrence days cannot be empty; use nil for one-off tasks")
	}

	seen := make(map[string]struct{}, len(days))

	for _, d := range days {
		if _, ok := validRecurrenceDays[d]; !ok {
			return fmt.Errorf("invalid recurrence day: %s", d)
		}

		if _, exists := seen[d]; exists {
			return fmt.Errorf("duplicate recurrence day: %s", d)
		}

		seen[d] = struct{}{}
	}

	return nil
}

func cloneStrings(v []string) []string {
	if v == nil {
		return nil
	}

	cp := make([]string, len(v))
	copy(cp, v)

	return cp
}
