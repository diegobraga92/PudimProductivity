package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

func TestReCompleteAfterUncomplete(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	service := task.NewTaskService(postgres.NewTaskRepository(pool), nil, nil)

	habit, err := service.CreateTask(ctx, "Morning run", []string{"mon", "wed", "fri"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	date := "2026-08-10" // a Monday in the past

	// 1. Check the habit for that date.
	if _, err := service.CompleteTask(ctx, habit.ID, date, nil); err != nil {
		t.Fatalf("first CompleteTask: %v", err)
	}

	// 2. Uncheck it (soft deletes the completion row).
	if err := service.UncompleteTask(ctx, habit.ID, date); err != nil {
		t.Fatalf("UncompleteTask: %v", err)
	}

	// 3. Re-checking the same date must succeed.
	if _, err := service.CompleteTask(ctx, habit.ID, date, nil); err != nil {
		t.Fatalf("re-CompleteTask after uncomplete: %v", err)
	}

	// 4. The re-created completion must be visible to reads.
	from := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	completions, err := service.GetTaskCompletions(ctx, habit.ID, from, to)
	if err != nil {
		t.Fatalf("GetTaskCompletions: %v", err)
	}
	if len(completions) != 1 {
		t.Fatalf("want exactly 1 active completion after re-complete, got %d", len(completions))
	}
	if got := completions[0].CompletedDate.Format("2006-01-02"); got != date {
		t.Fatalf("unexpected completion date: got %s, want %s", got, date)
	}
}

// TestDoubleCompleteStillConflicts preserves the API contract that completing a
// date with an already-active completion keeps failing with task.ErrCompletionAlreadyExists.
func TestDoubleCompleteStillConflicts(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	service := task.NewTaskService(postgres.NewTaskRepository(pool), nil, nil)

	habit, err := service.CreateTask(ctx, "Read 20 pages", []string{"mon", "tue"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	date := "2026-08-11"
	if _, err := service.CompleteTask(ctx, habit.ID, date, nil); err != nil {
		t.Fatalf("first CompleteTask: %v", err)
	}

	if _, err := service.CompleteTask(ctx, habit.ID, date, nil); err == nil {
		t.Fatal("second CompleteTask: want task.ErrCompletionAlreadyExists, got nil")
	} else if !errors.Is(err, task.ErrCompletionAlreadyExists) {
		t.Fatalf("second CompleteTask: want task.ErrCompletionAlreadyExists, got %v", err)
	}
}

// Positions are scoped per bucket, so a reorder of the to-dos must not touch
// habits and a stale/foreign id must fail the whole request.
func TestReorderScopedPerBucket(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	service := task.NewTaskService(postgres.NewTaskRepository(pool), nil, nil)

	first, err := service.CreateTask(ctx, "first", nil)
	if err != nil {
		t.Fatalf("CreateTask(first): %v", err)
	}
	second, err := service.CreateTask(ctx, "second", nil)
	if err != nil {
		t.Fatalf("CreateTask(second): %v", err)
	}
	habit, err := service.CreateTask(ctx, "habit", []string{"mon"})
	if err != nil {
		t.Fatalf("CreateTask(habit): %v", err)
	}

	if first.Position != 0 || second.Position != 1 || habit.Position != 0 {
		t.Fatalf("new tasks should append to their own bucket: first=%d second=%d habit=%d",
			first.Position, second.Position, habit.Position)
	}

	scope := task.TaskScope{Status: task.TaskStatusTodo}
	if err := service.ReorderTasks(ctx, scope, []string{second.ID, first.ID}); err != nil {
		t.Fatalf("ReorderTasks: %v", err)
	}

	got, err := service.GetTask(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Position != 1 {
		t.Errorf("first task position: got %d, want 1", got.Position)
	}

	// A habit id cannot be placed in the to-do bucket.
	if err := service.ReorderTasks(ctx, scope, []string{first.ID, habit.ID}); !errors.Is(err, task.ErrTaskScopeMismatch) {
		t.Fatalf("mixed buckets: got %v, want ErrTaskScopeMismatch", err)
	}

	// The rejected request must not have moved anything.
	got, err = service.GetTask(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetTask after rejected reorder: %v", err)
	}
	if got.Position != 1 {
		t.Errorf("position changed on a rejected reorder: got %d, want 1", got.Position)
	}
}
