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
	id := shared.NewUUID()

	task, err := NewTask(id, title, recurrenceDays)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

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

// ListTasks returns all tasks, optionally filtered by status.
func (s *TaskService) ListTasks(ctx context.Context, statusFilter string) ([]*Task, error) {
	return s.repo.List(ctx, statusFilter)
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

// CompleteTask marks a habit task as completed for today.
// Returns an error if the task is not a habit.
func (s *TaskService) CompleteTask(ctx context.Context, taskID string) (*TaskCompletion, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !task.IsHabit() {
		return nil, fmt.Errorf("task %s is not a habit and cannot be completed via completions", taskID)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	completion, err := NewTaskCompletion(shared.NewUUID(), taskID, today)
	if err != nil {
		return nil, fmt.Errorf("create completion: %w", err)
	}

	if err := s.repo.CreateCompletion(ctx, completion); err != nil {
		return nil, fmt.Errorf("persist completion: %w", err)
	}

	if err := s.bus.Publish(ctx, NewTaskCompleted(taskID, today)); err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("failed to publish TaskCompleted event")
	}

	log.Info().Str("task_id", taskID).Msg("task completed for today")
	return completion, nil
}

// UncompleteTask removes a habit task's completion for today.
func (s *TaskService) UncompleteTask(ctx context.Context, taskID string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if !task.IsHabit() {
		return fmt.Errorf("task %s is not a habit", taskID)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := s.repo.DeleteCompletion(ctx, taskID, today); err != nil {
		return err
	}

	if err := s.bus.Publish(ctx, NewTaskUncompleted(taskID, today)); err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("failed to publish TaskUncompleted event")
	}

	log.Info().Str("task_id", taskID).Msg("task uncompleted for today")
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
