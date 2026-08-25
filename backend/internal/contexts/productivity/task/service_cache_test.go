package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/cache"
)

// countingRepo is a minimal TaskRepository that records how many times List is
// called, so tests can prove read-through caching and invalidation work.
type countingRepo struct {
	tasks     []*Task
	listCalls atomic.Int32
}

func (r *countingRepo) Create(_ context.Context, t *Task) error {
	r.tasks = append(r.tasks, t)
	return nil
}
func (r *countingRepo) GetByID(_ context.Context, id string) (*Task, error) {
	for _, t := range r.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, ErrTaskNotFound
}
func (r *countingRepo) List(_ context.Context, _, _ string) ([]*Task, error) {
	r.listCalls.Add(1)
	return r.tasks, nil
}
func (r *countingRepo) ListScheduled(_ context.Context) ([]*Task, error) { return nil, nil }
func (r *countingRepo) ListByListID(_ context.Context, _, _ string) ([]*Task, error) {
	return nil, nil
}
func (r *countingRepo) Update(_ context.Context, _ *Task) error  { return nil }
func (r *countingRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *countingRepo) CreateCompletion(_ context.Context, _ *TaskCompletion) error {
	return nil
}
func (r *countingRepo) DeleteCompletion(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (r *countingRepo) GetCompletion(_ context.Context, _ string, _ time.Time) (*TaskCompletion, error) {
	return nil, nil
}
func (r *countingRepo) ListCompletions(_ context.Context, _ string, _, _ time.Time) ([]*TaskCompletion, error) {
	return nil, nil
}
func (r *countingRepo) ListAllCompletions(_ context.Context, _, _ time.Time) ([]*TaskCompletion, error) {
	return nil, nil
}

func newCachedService(t *testing.T, repo *countingRepo) *TaskService {
	t.Helper()
	s := miniredis.RunT(t)
	rc, err := cache.New(context.Background(), "redis://"+s.Addr(), time.Minute)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })
	return NewTaskService(repo, nil, nil, rc)
}

func TestListTasksReadThrough(t *testing.T) {
	repo := &countingRepo{tasks: []*Task{{ID: "1", Title: "A"}}}
	svc := newCachedService(t, repo)
	ctx := context.Background()

	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if got := repo.listCalls.Load(); got != 1 {
		t.Fatalf("List called %d times, want 1 (second read should be cached)", got)
	}
}

func TestListTasksInvalidatedOnCreate(t *testing.T) {
	repo := &countingRepo{}
	svc := newCachedService(t, repo)
	ctx := context.Background()

	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if _, err := svc.CreateTask(ctx, "New task", nil); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if got := repo.listCalls.Load(); got != 2 {
		t.Fatalf("List called %d times, want 2 (cache must be invalidated after create)", got)
	}
}

func TestListTasksWithoutCacheAlwaysHitsRepo(t *testing.T) {
	repo := &countingRepo{tasks: []*Task{{ID: "1", Title: "A"}}}
	svc := NewTaskService(repo, nil, nil) // no cache
	ctx := context.Background()

	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if _, err := svc.ListTasks(ctx, "", ""); err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if got := repo.listCalls.Load(); got != 2 {
		t.Fatalf("List called %d times, want 2 (no cache = always hit repo)", got)
	}
}
