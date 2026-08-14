package library

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func setupLibraryTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("pudimproductivity"),
		postgres.WithUsername("pudim"),
		postgres.WithPassword("pudim_dev"),
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

	pool, err := db.ConnectPool(ctx, shared.DatabaseConfig{
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

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

func TestLibraryRepository_CRUDAndFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)
	repo := NewPostgresRepository(pool)

	item, err := NewItem(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "The Matrix",
		MediaTypeMovie, ptrInt(1999), false, "Sci-fi classic", ptrFloat(8.7), "imdb", "Sci-fi",
	)
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "The Matrix" || got.MediaType != MediaTypeMovie ||
		got.ReleaseYear == nil || *got.ReleaseYear != 1999 || got.Done || got.Notes != "Sci-fi classic" ||
		got.Score == nil || *got.Score != 8.7 || got.ScoreSource != "imdb" {
		t.Fatalf("GetByID incomplete: %+v", got)
	}

	// List filters.
	movies, err := repo.List(ctx, "movie", nil)
	if err != nil || len(movies) != 1 {
		t.Fatalf("List(movie) = %v (err %v)", movies, err)
	}
	undone := false
	undoneList, err := repo.List(ctx, "", &undone)
	if err != nil || len(undoneList) != 1 {
		t.Fatalf("List(done=false) = %v (err %v)", undoneList, err)
	}
	done := true
	noneDone, err := repo.List(ctx, "", &done)
	if err != nil || len(noneDone) != 0 {
		t.Fatalf("List(done=true) = %v (err %v), want empty", noneDone, err)
	}

	// Update.
	item.Done = true
	if err := repo.Update(ctx, item); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.GetByID(ctx, item.ID)
	if err != nil || !got.Done {
		t.Fatalf("after update done = %v (err %v)", got, err)
	}

	// Delete.
	if err := repo.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID after delete: want ErrNotFound, got %v", err)
	}
}

func TestLibraryRepository_ImportBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)
	repo := NewPostgresRepository(pool)

	items := []*Item{
		{
			ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Name: "Breaking Bad", MediaType: MediaTypeSeries, ReleaseYear: ptrInt(2008), Notes: "",
		},
		{
			ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Name: "Elden Ring", MediaType: MediaTypeGame, ReleaseYear: ptrInt(2022), Notes: "GOTY",
		},
	}
	if err := repo.Import(ctx, items); err != nil {
		t.Fatalf("Import: %v", err)
	}

	all, err := repo.List(ctx, "", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List after import = %d rows, want 2", len(all))
	}
	for _, it := range all {
		if it.Name != "Breaking Bad" && it.Name != "Elden Ring" {
			t.Fatalf("unexpected item: %+v", it)
		}
	}
}

func TestLibraryRepository_ImportRollsBackOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)
	repo := NewPostgresRepository(pool)

	// Second item is invalid (empty name would still insert — use a duplicate
	// UUID to force a constraint violation mid-batch instead).
	dup := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	good, _ := NewItem(dup, "Good", MediaTypeMovie, nil, false, "", nil, "", "")
	if err := repo.Create(ctx, good); err != nil {
		t.Fatalf("seed: %v", err)
	}

	bad, _ := NewItem(dup, "Duplicate ID", MediaTypeGame, nil, false, "", nil, "", "")
	items := []*Item{
		{ID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", Name: "First", MediaType: MediaTypeBook, ReleaseYear: nil, Notes: ""},
		bad,
	}
	if err := repo.Import(ctx, items); err == nil {
		t.Fatal("expected Import to fail on duplicate id")
	}

	// The valid row before the failure must have been rolled back.
	all, err := repo.List(ctx, "", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List after failed import = %d rows, want 1 (rollback)", len(all))
	}
}
