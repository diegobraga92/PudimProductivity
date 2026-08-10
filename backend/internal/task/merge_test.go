package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRepo is a minimal in-memory TaskRepository used to test the LWW merge
// semantics without a database.
type fakeRepo struct {
	store map[string]*Task
}

func newFakeRepo(tasks ...*Task) *fakeRepo {
	r := &fakeRepo{store: map[string]*Task{}}
	for _, t := range tasks {
		r.store[t.ID] = t
	}
	return r
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (*Task, error) {
	t, ok := r.store[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t, nil
}

func (r *fakeRepo) Update(_ context.Context, t *Task) error {
	if _, ok := r.store[t.ID]; !ok {
		return ErrTaskNotFound
	}
	r.store[t.ID] = t
	return nil
}

func (r *fakeRepo) Create(_ context.Context, _ *Task) error              { return nil }
func (r *fakeRepo) List(_ context.Context, _, _ string) ([]*Task, error) { return nil, nil }
func (r *fakeRepo) ListScheduled(_ context.Context) ([]*Task, error)     { return nil, nil }
func (r *fakeRepo) ListByListID(_ context.Context, _, _ string) ([]*Task, error) {
	return nil, nil
}
func (r *fakeRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *fakeRepo) CreateCompletion(_ context.Context, _ *TaskCompletion) error {
	return nil
}
func (r *fakeRepo) DeleteCompletion(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (r *fakeRepo) GetCompletion(_ context.Context, _ string, _ time.Time) (*TaskCompletion, error) {
	return nil, nil
}
func (r *fakeRepo) ListCompletions(_ context.Context, _ string, _, _ time.Time) ([]*TaskCompletion, error) {
	return nil, nil
}
func (r *fakeRepo) ListAllCompletions(_ context.Context, _, _ time.Time) ([]*TaskCompletion, error) {
	return nil, nil
}

func testTaskWithUpdatedAt(t *testing.T, id, title string, updatedAt time.Time) *Task {
	t.Helper()
	task, err := NewTask(id, title, nil)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	task.UpdatedAt = updatedAt
	return task
}

// MergeTask with a newer client timestamp must win and persist.
func TestMergeTask_NewerWins(t *testing.T) {
	base := testTaskWithUpdatedAt(t, "t1", "old", time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC))
	svc := NewTaskService(newFakeRepo(base), nil, nil)

	newTitle := "new title"
	merged, applied, err := svc.MergeTask(context.Background(), "t1", "user-a",
		time.Date(2026, 8, 10, 11, 0, 0, 0, time.UTC),
		&newTitle, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("MergeTask: %v", err)
	}
	if !applied {
		t.Fatal("expected the newer write to win")
	}
	if merged.Title != "new title" {
		t.Errorf("Title: got %q, want %q", merged.Title, "new title")
	}
	if merged.UpdatedBy != "user-a" {
		t.Errorf("UpdatedBy: got %q, want %q", merged.UpdatedBy, "user-a")
	}
	if !merged.UpdatedAt.Equal(time.Date(2026, 8, 10, 11, 0, 0, 0, time.UTC)) {
		t.Errorf("UpdatedAt: got %v, want client timestamp", merged.UpdatedAt)
	}
}

// MergeTask with an older timestamp must lose and return the winning state.
func TestMergeTask_OlderLoses(t *testing.T) {
	base := testTaskWithUpdatedAt(t, "t1", "winner", time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))
	svc := NewTaskService(newFakeRepo(base), nil, nil)

	staleTitle := "stale write"
	merged, applied, err := svc.MergeTask(context.Background(), "t1", "user-a",
		time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC),
		&staleTitle, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("MergeTask: %v", err)
	}
	if applied {
		t.Fatal("expected the older write to lose")
	}
	if merged.Title != "winner" {
		t.Errorf("Title: got %q, want winning state %q", merged.Title, "winner")
	}
}

// Equal timestamps break ties by the lexicographically greater updated_by.
func TestMergeTask_EqualTimestampTieBreaksByUserID(t *testing.T) {
	ts := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	base := testTaskWithUpdatedAt(t, "t1", "from-b", ts)
	base.UpdatedBy = "user-b"
	svc := NewTaskService(newFakeRepo(base), nil, nil)

	fromA := "from-a"
	merged, applied, err := svc.MergeTask(context.Background(), "t1", "user-a", ts,
		&fromA, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("MergeTask: %v", err)
	}
	if applied {
		t.Fatal("user-a must lose the tie against user-b")
	}
	if merged.Title != "from-b" {
		t.Errorf("Title: got %q, want %q", merged.Title, "from-b")
	}

	// Reverse: user-z beats user-b on the same timestamp.
	svc2 := NewTaskService(newFakeRepo(base), nil, nil)
	fromZ := "from-z"
	merged2, applied2, err := svc2.MergeTask(context.Background(), "t1", "user-z", ts,
		&fromZ, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("MergeTask: %v", err)
	}
	if !applied2 {
		t.Fatal("user-z must win the tie against user-b")
	}
	if merged2.Title != "from-z" {
		t.Errorf("Title: got %q, want %q", merged2.Title, "from-z")
	}
}

// A zero client timestamp is stamped with the server clock and wins over any
// stored (past) state.
func TestMergeTask_ZeroTimestampAlwaysWins(t *testing.T) {
	base := testTaskWithUpdatedAt(t, "t1", "current",
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	svc := NewTaskService(newFakeRepo(base), nil, nil)

	newTitle := "forced"
	_, applied, err := svc.MergeTask(context.Background(), "t1", "user-a", time.Time{},
		&newTitle, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("MergeTask: %v", err)
	}
	if !applied {
		t.Fatal("expected a zero-timestamp write to win")
	}
}

func TestMergeTask_MissingTask(t *testing.T) {
	svc := NewTaskService(newFakeRepo(), nil, nil)
	title := "x"
	_, _, err := svc.MergeTask(context.Background(), "missing", "user-a", time.Now(),
		&title, nil, nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}
