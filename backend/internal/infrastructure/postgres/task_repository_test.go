package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	testpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

// setupTaskTestPostgres starts a fresh Postgres and applies all migrations.
func setupTaskTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := testpg.Run(ctx, "postgres:16-alpine",
		testpg.WithDatabase("pudimproductivity"),
		testpg.WithUsername("pudim"),
		testpg.WithPassword("pudim_dev"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := postgres.ConnectPool(ctx, config.DatabaseConfig{
		URL:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

// TestReCompleteAfterUncomplete guards against the Phase 9c regression where the
// soft-delete tombstone kept occupying the UNIQUE(task_id, completed_date) key,
// making a completion impossible to re-check after it was unchecked (false 409).
func TestReCompleteAfterUncomplete(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupTaskTestPostgres(t)
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

	// 3. Re-checking the same date must succeed — this was the bug: the
	//    tombstone row blocked re-insertion with a false "already completed".
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
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupTaskTestPostgres(t)
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
