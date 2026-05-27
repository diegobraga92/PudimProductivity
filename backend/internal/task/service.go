package task

import (
	"context"
	"fmt"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

// ErrTaskNotFound is returned when a task is not found in the repository.
var ErrTaskNotFound = fmt.Errorf("task not found")

// ErrCompletionAlreadyExists is returned when a habit completion already exists for the given task+date.
var ErrCompletionAlreadyExists = fmt.Errorf("completion already exists for this task on this date")

// ErrCompletionNotFound is returned when no completion exists for the given task+date.
var ErrCompletionNotFound = fmt.Errorf("completion not found")

// TaskService implements the application logic for task operations.
type TaskService struct {
	repo TaskRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(repo TaskRepository) *TaskService {
	return &TaskService{
		repo: repo,
	}
}

// CreateTask creates a new task.
func (s *TaskService) CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error) {
	return s.CreateTaskWithList(ctx, title, recurrenceDays, nil)
}

// CreateTaskWithList creates a new task assigned to a specific list.
func (s *TaskService) CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error) {
	id := shared.NewUUID()

	task, err := NewTask(id, title, recurrenceDays)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	task.ListID = listID

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task: %w", err)
	}

	log.Info().Str("task_id", task.ID).Str("title", task.Title).Msg("task created")
	return task, nil
}

// GetTask retrieves a task by its ID.
func (s *TaskService) GetTask(ctx context.Context, id string) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// ListTasks returns all tasks not assigned to any list, optionally filtered by status and type.
func (s *TaskService) ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	return s.repo.List(ctx, statusFilter, typeFilter)
}

// ListTasksByListID returns all tasks belonging to a specific task list, optionally filtered by type.
func (s *TaskService) ListTasksByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error) {
	return s.repo.ListByListID(ctx, listID, typeFilter)
}

// UpdateTask updates an existing task.
// listID uses double-pointer semantics: nil = no change, &nil = unassign, &ptr = assign.
func (s *TaskService) UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := task.Update(title, status, recurrenceDays, listID); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task update: %w", err)
	}

	log.Info().Str("task_id", task.ID).Msg("task updated")
	return task, nil
}

// DeleteTask deletes a task.
func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("task_id", id).Msg("task deleted")
	return nil
}

// CompleteTask marks a habit task as completed for a specific date.
// If dateStr is empty, defaults to today (UTC).
// Returns an error if the task is not a habit.
func (s *TaskService) CompleteTask(ctx context.Context, taskID, dateStr string) (*TaskCompletion, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !task.IsHabit() {
		return nil, fmt.Errorf("task %s is not a habit and cannot be completed via completions", taskID)
	}

	completionDate := time.Now().UTC().Truncate(24 * time.Hour)
	if dateStr != "" {
		completionDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format %q, expected YYYY-MM-DD: %w", dateStr, err)
		}
	}

	completion, err := NewTaskCompletion(shared.NewUUID(), taskID, completionDate)
	if err != nil {
		return nil, fmt.Errorf("create completion: %w", err)
	}

	if err := s.repo.CreateCompletion(ctx, completion); err != nil {
		return nil, fmt.Errorf("persist completion: %w", err)
	}

	log.Info().Str("task_id", taskID).Str("date", completionDate.Format("2006-01-02")).Msg("task completed")
	return completion, nil
}

// UncompleteTask removes a habit task's completion for a specific date.
// If dateStr is empty, defaults to today (UTC).
func (s *TaskService) UncompleteTask(ctx context.Context, taskID, dateStr string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if !task.IsHabit() {
		return fmt.Errorf("task %s is not a habit", taskID)
	}

	completionDate := time.Now().UTC().Truncate(24 * time.Hour)
	if dateStr != "" {
		completionDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format %q, expected YYYY-MM-DD: %w", dateStr, err)
		}
	}

	if err := s.repo.DeleteCompletion(ctx, taskID, completionDate); err != nil {
		return err
	}

	log.Info().Str("task_id", taskID).Str("date", completionDate.Format("2006-01-02")).Msg("task uncompleted")
	return nil
}

// GetTaskCompletions returns all completions for a task within a date range.
func (s *TaskService) GetTaskCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error) {
	// Verify the task exists
	_, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListCompletions(ctx, taskID, from, to)
}

// GetTodayCompletion returns the completion for a task today, or nil if not completed.
func (s *TaskService) GetTodayCompletion(ctx context.Context, taskID string) (*TaskCompletion, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return s.repo.GetCompletion(ctx, taskID, today)
}

// GetAllTaskCompletions returns all completions across all tasks within a date range.
// This is a batch alternative to calling GetTaskCompletions per task.
func (s *TaskService) GetAllTaskCompletions(ctx context.Context, from, to time.Time) ([]*TaskCompletion, error) {
	return s.repo.ListAllCompletions(ctx, from, to)
}
