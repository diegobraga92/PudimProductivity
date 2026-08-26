package task

import (
	"context"
	"fmt"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

var ErrTaskNotFound = fmt.Errorf("task not found")
var ErrCompletionAlreadyExists = fmt.Errorf("completion already exists for this task on this date")
var ErrCompletionNotFound = fmt.Errorf("completion not found")

// taskCacheNamespace is the cache namespace used for all task reads. Mutations
// bump its version so every previously cached task entry is invalidated.
const taskCacheNamespace = "tasks"

// TaskCache is the cache port used for read-through caching of task reads.
type TaskCache interface {
	// Get unmarshals the cached value into dest. Returns (true, nil) on a hit.
	Get(ctx context.Context, key string, dest any) (bool, error)
	// Set stores value under key with the cache's TTL.
	Set(ctx context.Context, key string, value any) error
	// Version returns the current invalidation version for a namespace.
	Version(ctx context.Context, ns string) (int64, error)
	// Bump increments the namespace version, invalidating cached entries.
	Bump(ctx context.Context, ns string) (int64, error)
}

type TaskService struct {
	repo  TaskRepository
	audit audit.Logger
	bus   eventbus.Bus
	cache TaskCache // nil = caching disabled
}

// NewTaskService builds the task service.
func NewTaskService(repo TaskRepository, auditLogger audit.Logger, bus eventbus.Bus, cache ...TaskCache) *TaskService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	s := &TaskService{
		repo:  repo,
		audit: auditLogger,
		bus:   bus,
	}
	if len(cache) > 0 && cache[0] != nil {
		s.cache = cache[0]
	}
	return s
}

// publish emits an event to the bus.
func (s *TaskService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish domain event")
	}
}

// invalidate bumps the task cache namespace after a mutation so no stale
// cached read is served. No-op when caching is disabled.
func (s *TaskService) invalidate(ctx context.Context) {
	if s.cache == nil {
		return
	}
	if _, err := s.cache.Bump(ctx, taskCacheNamespace); err != nil {
		log.Warn().Err(err).Msg("task cache: invalidation bump failed")
	}
}

// cacheKey prefixes a cache key with the current namespace version, so a bump
// atomically invalidates every previously cached task entry.
func (s *TaskService) cacheKey(ctx context.Context, inner string) string {
	version, err := s.cache.Version(ctx, taskCacheNamespace)
	if err != nil {
		log.Warn().Err(err).Msg("task cache: version read failed — treating as miss")
		version = 0
	}
	return fmt.Sprintf("tasks:v%d:%s", version, inner)
}

// cachedList is a read-through cache helper for slice-of-task reads.
func (s *TaskService) cachedList(ctx context.Context, inner string, load func() ([]*Task, error)) ([]*Task, error) {
	if s.cache == nil {
		return load()
	}
	key := s.cacheKey(ctx, inner)
	var cached []*Task
	if hit, err := s.cache.Get(ctx, key, &cached); err == nil && hit {
		return cached, nil
	} else if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("task cache: get failed — falling through to repo")
	}
	tasks, err := load()
	if err != nil {
		return nil, err
	}
	if err := s.cache.Set(ctx, key, tasks); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("task cache: set failed")
	}
	return tasks, nil
}

// cachedTask is a read-through cache helper for single-task reads.
func (s *TaskService) cachedTask(ctx context.Context, id string, load func() (*Task, error)) (*Task, error) {
	if s.cache == nil {
		return load()
	}
	key := s.cacheKey(ctx, "task:"+id)
	var cached *Task
	if hit, err := s.cache.Get(ctx, key, &cached); err == nil && hit {
		return cached, nil
	} else if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("task cache: get failed — falling through to repo")
	}
	task, err := load()
	if err != nil {
		return nil, err
	}
	if err := s.cache.Set(ctx, key, task); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("task cache: set failed")
	}
	return task, nil
}

func (s *TaskService) CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error) {
	return s.CreateTaskWithSchedule(ctx, title, recurrenceDays, nil, nil, nil, nil, nil, nil, nil)
}

func (s *TaskService) CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error) {
	return s.CreateTaskWithSchedule(ctx, title, recurrenceDays, listID, nil, nil, nil, nil, nil, nil)
}

func (s *TaskService) CreateTaskWithSchedule(ctx context.Context, title string, recurrenceDays []string, listID *string, startTime, endTime, color, scheduledDate *string, alarmMinutes *int, clientID *string) (*Task, error) {
	id := uuid.NewUUID()
	if clientID != nil && *clientID != "" {
		// Offline-first clients generate their own UUIDs so a create is idempotent.
		if _, err := guuid.Parse(*clientID); err != nil {
			return nil, fmt.Errorf("create task: invalid id: %w", err)
		}
		id = *clientID
		if existing, err := s.repo.GetByID(ctx, id); err == nil {
			return existing, nil
		}
	}

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

	s.invalidate(ctx)
	s.publish(ctx, eventbus.EventTaskCreated, ToTaskResponse(task))

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*Task, error) {
	return s.cachedTask(ctx, id, func() (*Task, error) { return s.repo.GetByID(ctx, id) })
}

func (s *TaskService) ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	inner := "list:" + statusFilter + "|" + typeFilter
	return s.cachedList(ctx, inner, func() ([]*Task, error) { return s.repo.List(ctx, statusFilter, typeFilter) })
}

func (s *TaskService) ListScheduledTasks(ctx context.Context) ([]*Task, error) {
	return s.cachedList(ctx, "scheduled", func() ([]*Task, error) { return s.repo.ListScheduled(ctx) })
}

func (s *TaskService) ListTasksByListID(ctx context.Context, listID, typeFilter string) ([]*Task, error) {
	inner := "byList:" + listID + "|" + typeFilter
	return s.cachedList(ctx, inner, func() ([]*Task, error) { return s.repo.ListByListID(ctx, listID, typeFilter) })
}

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

	s.invalidate(ctx)
	s.publish(ctx, eventbus.EventTaskUpdated, ToTaskResponse(task))

	return task, nil
}

// MergeTask applies a client-authored update using document-level LWW (last-writer-wins) semantics.
func (s *TaskService) MergeTask(ctx context.Context, id, userID string, clientUpdatedAt time.Time, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, bool, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if clientUpdatedAt.IsZero() {
		clientUpdatedAt = time.Now().UTC()
	}

	wins := clientUpdatedAt.After(current.UpdatedAt) ||
		(clientUpdatedAt.Equal(current.UpdatedAt) && (current.UpdatedBy == nil || userID > *current.UpdatedBy))
	if !wins {
		return current, false, nil
	}

	if err := current.Update(title, status, recurrenceDays, listID, startTime, endTime, color, scheduledDate, alarmMinutes); err != nil {
		return nil, false, fmt.Errorf("merge task: %w", err)
	}
	// The merge carries the client timestamp (not the server clock) so
	// concurrent writers can compare against a stable value.
	current.UpdatedAt = clientUpdatedAt
	current.UpdatedBy = &userID

	if err := s.repo.Update(ctx, current); err != nil {
		return nil, false, fmt.Errorf("persist task merge: %w", err)
	}

	log.Info().Ctx(ctx).Str("task_id", current.ID).Str("by", userID).Msg("task merged")
	s.audit.Log(ctx, audit.ActionTaskUpdated, audit.ResourceTasks, current.ID, nil, map[string]any{"merged": true})
	s.invalidate(ctx)
	s.publish(ctx, eventbus.EventTaskMerged, ToTaskResponse(current))

	return current, true, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Ctx(ctx).Str("task_id", id).Msg("task deleted")

	s.audit.Log(ctx, audit.ActionTaskDeleted, audit.ResourceTasks, id, nil, nil)

	s.invalidate(ctx)
	s.publish(ctx, eventbus.EventTaskDeleted, map[string]any{"id": id})

	return nil
}

func (s *TaskService) CompleteTask(ctx context.Context, taskID, dateStr string, completionID *string) (*TaskCompletion, error) {
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

	id := uuid.NewUUID()
	if completionID != nil && *completionID != "" {
		// Offline-first clients generate their own completion UUIDs for idempotency.
		if _, err := guuid.Parse(*completionID); err != nil {
			return nil, fmt.Errorf("complete task: invalid id: %w", err)
		}
		id = *completionID
		if existing, err := s.repo.GetCompletion(ctx, taskID, completionDate); err == nil && existing != nil {
			if existing.ID == id {
				return existing, nil
			}
			return nil, ErrCompletionAlreadyExists
		}
	}

	completion, err := NewTaskCompletion(id, taskID, completionDate)
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

	s.invalidate(ctx)
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

	s.invalidate(ctx)
	s.publish(ctx, eventbus.EventTaskUncompleted, map[string]any{
		"id":             taskID,
		"title":          task.Title,
		"completed_date": completionDate.Format("2006-01-02"),
	})

	return nil
}

func (s *TaskService) GetTaskCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*TaskCompletion, error) {
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
