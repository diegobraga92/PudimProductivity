package task

import (
	"context"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

type CachedTaskService struct {
	*TaskService
	cache *shared.Cache
}

func NewCachedTaskService(service *TaskService, cache *shared.Cache) *CachedTaskService {
	return &CachedTaskService{
		TaskService: service,
		cache:       cache,
	}
}

// Cache key: tasks:list?status={status}&type={type}
func (s *CachedTaskService) ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*Task, error) {
	cacheKey := "tasks:list"
	if statusFilter != "" {
		cacheKey += "?status=" + statusFilter
	}
	if typeFilter != "" {
		cacheKey += "?type=" + typeFilter
	}

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
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.cache.Set(cacheCtx, cacheKey, tasks); err != nil {
			log.Warn().Err(err).Str("cache_key", cacheKey).Msg("failed to cache tasks list")
		}
	}()

	return tasks, nil
}

func (s *CachedTaskService) GetTask(ctx context.Context, id string) (*Task, error) {
	cacheKey := shared.Key(shared.CacheKeyTask, id)

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
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
	s.invalidateAll(ctx)
	return task, nil
}

func (s *CachedTaskService) CreateTaskWithList(ctx context.Context, title string, recurrenceDays []string, listID *string) (*Task, error) {
	task, err := s.TaskService.CreateTaskWithList(ctx, title, recurrenceDays, listID)
	if err != nil {
		return nil, err
	}
	s.invalidateAll(ctx)
	return task, nil
}

func (s *CachedTaskService) UpdateTask(ctx context.Context, id string, title *string, status *TaskStatus, recurrenceDays *[]string, listID **string) (*Task, error) {
	task, err := s.TaskService.UpdateTask(ctx, id, title, status, recurrenceDays, listID)
	if err != nil {
		return nil, err
	}
	s.invalidateAll(ctx, id)
	return task, nil
}

func (s *CachedTaskService) DeleteTask(ctx context.Context, id string) error {
	if err := s.TaskService.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.invalidateAll(ctx, id)
	return nil
}

func (s *CachedTaskService) CompleteTask(ctx context.Context, taskID, dateStr string) (*TaskCompletion, error) {
	comp, err := s.TaskService.CompleteTask(ctx, taskID, dateStr)
	if err != nil {
		return nil, err
	}
	s.invalidateAll(ctx, taskID)
	return comp, nil
}

func (s *CachedTaskService) UncompleteTask(ctx context.Context, taskID, dateStr string) error {
	if err := s.TaskService.UncompleteTask(ctx, taskID, dateStr); err != nil {
		return err
	}
	s.invalidateAll(ctx, taskID)
	return nil
}

func (s *CachedTaskService) invalidateAll(ctx context.Context, taskIDs ...string) {
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.cache.Del(cacheCtx, "tasks:list"); err != nil {
			log.Warn().Err(err).Msg("failed to invalidate tasks list cache")
		}
		for _, id := range taskIDs {
			key := shared.Key(shared.CacheKeyTask, id)
			if err := s.cache.Del(cacheCtx, key); err != nil {
				log.Warn().Err(err).Str("cache_key", key).Msg("failed to invalidate task cache")
			}
		}
	}()
}
