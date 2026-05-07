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
func (s *TaskService) CreateTask(ctx context.Context, title string, description *string, priority TaskPriority, dueDate *time.Time) (*Task, error) {
	id := shared.NewUUID()

	task, err := NewTask(id, title, description, priority, dueDate)
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

// ListTasks returns all tasks, optionally filtered.
func (s *TaskService) ListTasks(ctx context.Context, statusFilter, priorityFilter string) ([]*Task, error) {
	return s.repo.List(ctx, statusFilter, priorityFilter)
}

// UpdateTask updates an existing task and publishes events.
func (s *TaskService) UpdateTask(ctx context.Context, id string, title *string, description *string, status *TaskStatus, priority *TaskPriority, dueDate **time.Time) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := task.Status

	if err := task.Update(title, description, status, priority, dueDate); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task update: %w", err)
	}

	// Publish events
	if err := s.bus.Publish(ctx, NewTaskUpdated(task)); err != nil {
		log.Warn().Err(err).Str("task_id", task.ID).Msg("failed to publish TaskUpdated event")
	}

	if oldStatus != task.Status {
		if err := s.bus.Publish(ctx, NewTaskStatusChanged(task, oldStatus)); err != nil {
			log.Warn().Err(err).Str("task_id", task.ID).Msg("failed to publish TaskStatusChanged event")
		}
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
