package task

import (
	"context"
	"fmt"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

var ErrTaskNotFound = fmt.Errorf("task not found")
var ErrCompletionAlreadyExists = fmt.Errorf("completion already exists for this task on this date")
var ErrCompletionNotFound = fmt.Errorf("completion not found")

type TaskService struct {
	repo  TaskRepository
	audit audit.Logger
	bus   eventbus.Bus
}

func NewTaskService(repo TaskRepository, auditLogger audit.Logger, bus eventbus.Bus) *TaskService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &TaskService{
		repo:  repo,
		audit: auditLogger,
		bus:   bus,
	}
}

// publish emits an event to the bus. A nil bus (tests, or degraded startup) is a
// no-op. Event publication is best-effort: a failure is logged, not propagated,
// so the core operation is never rolled back because the bus hiccuped.
func (s *TaskService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish domain event")
	}
}

func (s *TaskService) CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error) {
	return s.CreateTaskWithSchedule(ctx, title, recurrenceDays, nil, nil, nil, nil, nil, nil)
}

func (s *TaskService) CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error) {
	return s.CreateTaskWithSchedule(ctx, title, recurrenceDays, listID, nil, nil, nil, nil, nil)
}

func (s *TaskService) CreateTaskWithSchedule(ctx context.Context, title string, recurrenceDays []string, listID *string, startTime, endTime, color, scheduledDate *string, alarmMinutes *int) (*Task, error) {
	id := shared.NewUUID()

	task, err := NewTaskWithSchedule(id, title, recurrenceDays, startTime, endTime, color, scheduledDate, alarmMinutes)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	task.ListID = listID

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task: %w", err)
	}

	log.Info().Ctx(ctx).Str("task_id", task.ID).Str("title", task.Title).Msg("task created")

	s.audit.Log(ctx, audit.ActionTaskCreated, audit.ResourceTasks, task.ID, nil, map[string]any{
		"title":   task.Title,
		"list_id": task.ListID,
	})

	s.publish(ctx, eventbus.EventTaskCreated, ToTaskResponse(task))

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

func (s *TaskService) ListScheduledTasks(ctx context.Context) ([]*Task, error) {
	return s.repo.ListScheduled(ctx)
}

func (s *TaskService) ListTasksByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error) {
	return s.repo.ListByListID(ctx, listID, typeFilter)
}

// listID uses double-pointer semantics: nil = no change, &nil = unassign, &ptr = assign.
// startTime, endTime, color, scheduledDate also use double-pointer:
// nil = no change, &nil = unset, &ptr = assign value.
func (s *TaskService) UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := task.Update(title, status, recurrenceDays, listID, startTime, endTime, color, scheduledDate, alarmMinutes); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task update: %w", err)
	}

	log.Info().Ctx(ctx).Str("task_id", task.ID).Msg("task updated")

	s.audit.Log(ctx, audit.ActionTaskUpdated, audit.ResourceTasks, task.ID, nil, map[string]any{
		"title":  task.Title,
		"status": task.Status,
	})

	s.publish(ctx, eventbus.EventTaskUpdated, ToTaskResponse(task))

	return task, nil
}

// MergeTask applies a client-authored update using document-level LWW
// (last-writer-wins) semantics (Phase 8, ADR 010):
//
//   - If clientUpdatedAt is zero (client did not send a timestamp), the write
//     is stamped with the server clock and always wins.
//   - Otherwise the write wins when clientUpdatedAt is strictly newer than the
//     stored updated_at, or when the timestamps are equal AND the client user
//     ID sorts after the stored updated_by (deterministic tie-break).
//
// On a win the task is persisted, a task.merged event is published, and
// applied=true is returned. On a loss the current winning state is returned
// with applied=false so the client can reconcile and retry.
func (s *TaskService) MergeTask(ctx context.Context, id, userID string, clientUpdatedAt time.Time, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, bool, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if clientUpdatedAt.IsZero() {
		clientUpdatedAt = time.Now().UTC()
	}

	wins := clientUpdatedAt.After(current.UpdatedAt) ||
		(clientUpdatedAt.Equal(current.UpdatedAt) && userID > current.UpdatedBy)
	if !wins {
		return current, false, nil
	}

	if err := current.Update(title, status, recurrenceDays, listID, startTime, endTime, color, scheduledDate, alarmMinutes); err != nil {
		return nil, false, fmt.Errorf("merge task: %w", err)
	}
	// The merge carries the client timestamp (not the server clock) so
	// concurrent writers can compare against a stable value.
	current.UpdatedAt = clientUpdatedAt
	current.UpdatedBy = userID

	if err := s.repo.Update(ctx, current); err != nil {
		return nil, false, fmt.Errorf("persist task merge: %w", err)
	}

	log.Info().Ctx(ctx).Str("task_id", current.ID).Str("by", userID).Msg("task merged")
	s.audit.Log(ctx, audit.ActionTaskUpdated, audit.ResourceTasks, current.ID, nil, map[string]any{"merged": true})
	s.publish(ctx, eventbus.EventTaskMerged, ToTaskResponse(current))

	return current, true, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Ctx(ctx).Str("task_id", id).Msg("task deleted")

	s.audit.Log(ctx, audit.ActionTaskDeleted, audit.ResourceTasks, id, nil, nil)

	s.publish(ctx, eventbus.EventTaskDeleted, map[string]any{"id": id})

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

	log.Info().Ctx(ctx).Str("task_id", taskID).Str("date", completionDate.Format("2006-01-02")).Msg("task completed")

	s.audit.Log(ctx, audit.ActionTaskCompleted, audit.ResourceTasks, taskID, nil, map[string]any{
		"completed_date": completionDate.Format("2006-01-02"),
	})

	s.publish(ctx, eventbus.EventTaskCompleted, map[string]any{
		"id":             completion.ID,
		"task_id":        taskID,
		"title":          task.Title,
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

	log.Info().Ctx(ctx).Str("task_id", taskID).Str("date", completionDate.Format("2006-01-02")).Msg("task uncompleted")

	s.audit.Log(ctx, audit.ActionTaskUncompleted, audit.ResourceTasks, taskID, nil, nil)

	s.publish(ctx, eventbus.EventTaskUncompleted, map[string]any{
		"id":             taskID,
		"title":          task.Title,
		"completed_date": completionDate.Format("2006-01-02"),
	})

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
