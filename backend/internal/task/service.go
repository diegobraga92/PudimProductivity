package task

import (
	"context"
	"fmt"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

var ErrTaskNotFound = fmt.Errorf("task not found")
var ErrCompletionAlreadyExists = fmt.Errorf("completion already exists for this task on this date")
var ErrCompletionNotFound = fmt.Errorf("completion not found")

type TaskService struct {
	repo  TaskRepository
	audit audit.Logger
}

func NewTaskService(repo TaskRepository, auditLogger audit.Logger) *TaskService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &TaskService{
		repo:  repo,
		audit: auditLogger,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error) {
	return s.CreateTaskWithList(ctx, title, recurrenceDays, nil)
}

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

	s.audit.Log(ctx, audit.ActionTaskCreated, audit.ResourceTasks, task.ID, nil, map[string]any{
		"title":  task.Title,
		"list_id": task.ListID,
	})

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	return s.repo.List(ctx, statusFilter, typeFilter)
}

func (s *TaskService) ListTasksByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error) {
	return s.repo.ListByListID(ctx, listID, typeFilter)
}

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

	s.audit.Log(ctx, audit.ActionTaskUpdated, audit.ResourceTasks, task.ID, nil, map[string]any{
		"title":  task.Title,
		"status": task.Status,
	})

	return task, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("task_id", id).Msg("task deleted")

	s.audit.Log(ctx, audit.ActionTaskDeleted, audit.ResourceTasks, id, nil, nil)

	return nil
}

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

	s.audit.Log(ctx, audit.ActionTaskCompleted, audit.ResourceTasks, taskID, nil, map[string]any{
		"completed_date": completionDate.Format("2006-01-02"),
	})

	return completion, nil
}

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

	s.audit.Log(ctx, audit.ActionTaskUncompleted, audit.ResourceTasks, taskID, nil, nil)

	return nil
}

func (s *TaskService) GetTaskCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error) {
	// Verify the task exists
	_, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListCompletions(ctx, taskID, from, to)
}

func (s *TaskService) GetTodayCompletion(ctx context.Context, taskID string) (*TaskCompletion, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return s.repo.GetCompletion(ctx, taskID, today)
}

func (s *TaskService) GetAllTaskCompletions(ctx context.Context, from, to time.Time) ([]*TaskCompletion, error) {
	return s.repo.ListAllCompletions(ctx, from, to)
}
