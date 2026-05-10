package tasklist

import (
	"context"
	"fmt"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

// TaskListService implements the application logic for task list operations.
type TaskListService struct {
	repo TaskListRepository
}

// NewTaskListService creates a new TaskListService.
func NewTaskListService(repo TaskListRepository) *TaskListService {
	return &TaskListService{repo: repo}
}

// CreateTaskList creates a new task list.
func (s *TaskListService) CreateTaskList(ctx context.Context, name string) (*TaskList, error) {
	id := shared.NewUUID()

	list, err := NewTaskList(id, name)
	if err != nil {
		return nil, fmt.Errorf("create task list: %w", err)
	}

	if err := s.repo.Create(ctx, list); err != nil {
		return nil, fmt.Errorf("persist task list: %w", err)
	}

	log.Info().Str("list_id", list.ID).Str("name", list.Name).Msg("task list created")
	return list, nil
}

// GetTaskList retrieves a task list by its ID.
func (s *TaskListService) GetTaskList(ctx context.Context, id string) (*TaskList, error) {
	list, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListTaskLists returns all task lists.
func (s *TaskListService) ListTaskLists(ctx context.Context) ([]*TaskList, error) {
	return s.repo.List(ctx)
}

// UpdateTaskList updates an existing task list.
func (s *TaskListService) UpdateTaskList(ctx context.Context, id string, name, description *string) (*TaskList, error) {
	list, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := list.Update(name, description); err != nil {
		return nil, fmt.Errorf("update task list: %w", err)
	}

	if err := s.repo.Update(ctx, list); err != nil {
		return nil, fmt.Errorf("persist task list update: %w", err)
	}

	log.Info().Str("list_id", list.ID).Msg("task list updated")
	return list, nil
}

// DeleteTaskList deletes a task list.
func (s *TaskListService) DeleteTaskList(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("list_id", id).Msg("task list deleted")
	return nil
}
