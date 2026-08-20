package task

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/cache"
)

// TODO: Add wiring

const cacheOpTimeout = 3 * time.Second

type CachedTaskService struct {
	*TaskService
	cache *cache.Cache
}

func NewCachedTaskService(service *TaskService, cache *cache.Cache) *CachedTaskService {
	return &CachedTaskService{
		TaskService: service,
		cache:       cache,
	}
}

func (s *CachedTaskService) ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	const cacheKey = "tasks:list"

	var cached []*Task
	found, err := s.cache.Get(ctx, cacheKey, &cached)
	if err == nil && found {
		log.Debug().Str("cache_key", cacheKey).Int("count", len(cached)).Msg("cache hit for tasks list")
		return cached, nil
	}

	tasks, err := s.TaskService.ListTasks(ctx, statusFilter, typeFilter)
	if err != nil {
		return nil, err
	}

	// Populate cache asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()
		if err := s.cache.Set(cacheCtx, cacheKey, tasks); err != nil {
			log.Warn().Err(err).Str("cache_key", cacheKey).Msg("failed to cache tasks list")
		}
	}()

	return tasks, nil
}

func (s *CachedTaskService) GetTask(ctx context.Context, id string) (*Task, error) {
	cacheKey := cache.Key(cache.CacheKeyTask, id)

	var cached Task
	found, err := s.cache.Get(ctx, cacheKey, &cached)
	if err == nil && found {
		log.Debug().Str("cache_key", cacheKey).Msg("cache hit for task")
		return &cached, nil
	}

	task, err := s.TaskService.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
		defer cancel()
		if err := s.cache.Set(cacheCtx, cacheKey, task); err != nil {
			log.Warn().Err(err).Str("cache_key", cacheKey).Msg("failed to cache task")
		}
	}()

	return task, nil
}

func (s *CachedTaskService) CreateTask(ctx context.Context, title string, recurrenceDays []string) (*Task, error) {
	task, err := s.TaskService.CreateTask(ctx, title, recurrenceDays)
	if err != nil {
		return nil, err
	}
	s.invalidateList()
	return task, nil
}

func (s *CachedTaskService) CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error) {
	task, err := s.TaskService.CreateTaskWithList(ctx, title, recurrenceDays, listID)
	if err != nil {
		return nil, err
	}
	s.invalidateList()
	return task, nil
}

func (s *CachedTaskService) UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string, startTime, endTime, color, scheduledDate **string, alarmMinutes **int) (*Task, error) {
	task, err := s.TaskService.UpdateTask(ctx, id, title, status, recurrenceDays, listID, startTime, endTime, color, scheduledDate, alarmMinutes)
	if err != nil {
		return nil, err
	}
	s.invalidateList()
	s.invalidateTask(id)
	return task, nil
}

func (s *CachedTaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.TaskService.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.invalidateList()
	s.invalidateTask(id)
	return nil
}

func (s *CachedTaskService) CompleteTask(ctx context.Context, taskID, dateStr string, completionID *string) (*TaskCompletion, error) {
	comp, err := s.TaskService.CompleteTask(ctx, taskID, dateStr, completionID)
	if err != nil {
		return nil, err
	}
	s.invalidateList()
	return comp, nil
}

func (s *CachedTaskService) UncompleteTask(ctx context.Context, taskID, dateStr string) error {
	if err := s.TaskService.UncompleteTask(ctx, taskID, dateStr); err != nil {
		return err
	}
	s.invalidateList()
	return nil
}

// Uses write-invalidate pattern, since we're unsure of what's stale after changes
func (s *CachedTaskService) invalidateList() {
	cacheCtx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	if err := s.cache.Del(cacheCtx, "tasks:list"); err != nil {
		log.Warn().Err(err).Msg("failed to invalidate tasks list cache")
	}
}

func (s *CachedTaskService) invalidateTask(id string) {
	cacheCtx, cancel := context.WithTimeout(context.Background(), cacheOpTimeout)
	defer cancel()
	key := cache.Key(cache.CacheKeyTask, id)
	if err := s.cache.Del(cacheCtx, key); err != nil {
		log.Warn().Err(err).Str("cache_key", key).Msg("failed to invalidate task cache")
	}
}
