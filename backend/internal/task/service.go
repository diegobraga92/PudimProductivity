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

// TaskService implements the application logic for task operations.
type TaskService struct {
	repo TaskRepository
	bus  shared.EventBus
}

// NewTaskService creates a new TaskService.
func NewTaskService(repo TaskRepository, bus shared.EventBus) *TaskService {
	return &TaskService{
		repo: repo,
		bus:  bus,
	}
}

// CreateTask creates a new task and publishes a TaskCreated event.
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

	if err := s.bus.Publish(ctx, NewTaskCreated(task)); err != nil {
		log.Warn().Err(err).Str("task_id", task.ID).Msg("failed to publish TaskCreated event")
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

// UpdateTask updates an existing task and publishes events.
func (s *TaskService) UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := task.Update(title, status, recurrenceDays); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task update: %w", err)
	}

	if err := s.bus.Publish(ctx, NewTaskUpdated(task)); err != nil {
		log.Warn().Err(err).Str("task_id", task.ID).Msg("failed to publish TaskUpdated event")
	}

	log.Info().Str("task_id", task.ID).Msg("task updated")
	return task, nil
}

// DeleteTask deletes a task and publishes a TaskDeleted event.
func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.bus.Publish(ctx, NewTaskDeleted(id)); err != nil {
		log.Warn().Err(err).Str("task_id", id).Msg("failed to publish TaskDeleted event")
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

	if err := s.bus.Publish(ctx, NewTaskCompleted(taskID, completionDate)); err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("failed to publish TaskCompleted event")
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

	if err := s.bus.Publish(ctx, NewTaskUncompleted(taskID, completionDate)); err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("failed to publish TaskUncompleted event")
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
