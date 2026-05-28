package tasklist

import (
	"context"
	"fmt"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

type TaskListService struct {
	repo TaskListRepository
}

func NewTaskListService(repo TaskListRepository) *TaskListService {
	return &TaskListService{repo: repo}
}

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

func (s *TaskListService) GetTaskList(ctx context.Context, id string) (*TaskList, error) {
	list, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (s *TaskListService) ListTaskLists(ctx context.Context) ([]*TaskList, error) {
	return s.repo.List(ctx)
}

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

func (s *TaskListService) DeleteTaskList(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("list_id", id).Msg("task list deleted")
	return nil
}
