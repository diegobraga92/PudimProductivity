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

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

func setupLibraryTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func TestLibraryRepository_CRUDAndFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)
	repo := postgres.NewLibraryRepository(pool)

	item, err := library.NewItem(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "The Matrix",
		library.MediaTypeMovie, ptrInt(1999), false, "Sci-fi classic", ptrFloat(8.7), "imdb", "Sci-fi",
	)
	if err != nil {
		t.Fatalf("library.NewItem: %v", err)
	}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "The Matrix" || got.MediaType != library.MediaTypeMovie ||
		got.ReleaseYear == nil || *got.ReleaseYear != 1999 || got.Done || got.Notes != "Sci-fi classic" ||
		got.Subtype != "Sci-fi" || got.Score == nil || *got.Score != 8.7 || got.ScoreSource != "imdb" {
		t.Fatalf("GetByID incomplete: %+v", got)
	}

	// List filters.
	movies, err := repo.List(ctx, library.ListFilter{MediaType: "movie"})
	if err != nil || len(movies) != 1 {
		t.Fatalf("List(movie) = %v (err %v)", movies, err)
	}
	undone := false
	undoneList, err := repo.List(ctx, library.ListFilter{Done: &undone})
	if err != nil || len(undoneList) != 1 {
		t.Fatalf("List(done=false) = %v (err %v)", undoneList, err)
	}
	done := true
	noneDone, err := repo.List(ctx, library.ListFilter{Done: &done})
	if err != nil || len(noneDone) != 0 {
		t.Fatalf("List(done=true) = %v (err %v), want empty", noneDone, err)
	}
	// Subtype filter is exact and case-insensitive.
	scifi, err := repo.List(ctx, library.ListFilter{Subtype: "sci-fi"})
	if err != nil || len(scifi) != 1 {
		t.Fatalf("List(subtype=sci-fi) = %v (err %v), want 1", scifi, err)
	}
	scifiMovies, err := repo.List(ctx, library.ListFilter{MediaType: "movie", Subtype: "Sci-fi"})
	if err != nil || len(scifiMovies) != 1 {
		t.Fatalf("List(movie, subtype) = %v (err %v), want 1", scifiMovies, err)
	}
	noMatch, err := repo.List(ctx, library.ListFilter{MediaType: "game", Subtype: "Sci-fi"})
	if err != nil || len(noMatch) != 0 {
		t.Fatalf("List(game, subtype) = %v (err %v), want empty", noMatch, err)
	}

	// Distinct subtypes (genres/consoles) for the filter dropdown.
	subtypes, err := repo.DistinctSubtypes(ctx, "")
	if err != nil || len(subtypes) != 1 || subtypes[0] != "Sci-fi" {
		t.Fatalf("DistinctSubtypes = %v (err %v), want [Sci-fi]", subtypes, err)
	}
	movieSubtypes, err := repo.DistinctSubtypes(ctx, "movie")
	if err != nil || len(movieSubtypes) != 1 {
		t.Fatalf("DistinctSubtypes(movie) = %v (err %v), want [Sci-fi]", movieSubtypes, err)
	}
	gameSubtypes, err := repo.DistinctSubtypes(ctx, "game")
	if err != nil || len(gameSubtypes) != 0 {
		t.Fatalf("DistinctSubtypes(game) = %v (err %v), want empty", gameSubtypes, err)
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
	if _, err := repo.GetByID(ctx, item.ID); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("GetByID after delete: want library.ErrNotFound, got %v", err)
	}
}

func TestLibraryRepository_ImportBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)
	repo := postgres.NewLibraryRepository(pool)

	items := []*library.Item{
		{
			ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Name: "Breaking Bad", MediaType: library.MediaTypeSeries, ReleaseYear: ptrInt(2008), Notes: "",
		},
		{
			ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Name: "Elden Ring", MediaType: library.MediaTypeGame, ReleaseYear: ptrInt(2022), Notes: "GOTY",
		},
	}
	if err := repo.Import(ctx, items); err != nil {
		t.Fatalf("Import: %v", err)
	}

	all, err := repo.List(ctx, library.ListFilter{})
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
	repo := postgres.NewLibraryRepository(pool)

	// Second item is invalid (empty name would still insert — use a duplicate
	// UUID to force a constraint violation mid-batch instead).
	dup := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	good, _ := library.NewItem(dup, "Good", library.MediaTypeMovie, nil, false, "", nil, "", "")
	if err := repo.Create(ctx, good); err != nil {
		t.Fatalf("seed: %v", err)
	}

	bad, _ := library.NewItem(dup, "Duplicate ID", library.MediaTypeGame, nil, false, "", nil, "", "")
	items := []*library.Item{
		{ID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", Name: "First", MediaType: library.MediaTypeBook, ReleaseYear: nil, Notes: ""},
		bad,
	}
	if err := repo.Import(ctx, items); err == nil {
		t.Fatal("expected Import to fail on duplicate id")
	}

	// The valid row before the failure must have been rolled back.
	all, err := repo.List(ctx, library.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List after failed import = %d rows, want 1 (rollback)", len(all))
	}
}

// TestLibraryRepository_ScoreSourceNotNull guards the Library-page regression
// where GET /api/v1/library returned 500 ("can't scan into dest[8] (col:
// score_source): cannot scan NULL into *string"). Migration 022 declared
// score_source TEXT NULL while the Go domain models it as a non-nullable string
// (empty = no source), so rows written before migration 022 broke the list
// query. Migration 025 backfills NULLs and enforces NOT NULL DEFAULT ”.
func TestLibraryRepository_ScoreSourceNotNull(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupLibraryTestPostgres(t)

	var isNullable string
	var columnDefault *string
	err := pool.QueryRow(ctx, `
		SELECT is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'library_items' AND column_name = 'score_source'
	`).Scan(&isNullable, &columnDefault)
	if err != nil {
		t.Fatalf("query score_source column metadata: %v", err)
	}
	if isNullable != "NO" {
		t.Fatalf("score_source is_nullable = %q, want NO (migration 025 must keep it NOT NULL)", isNullable)
	}
	if columnDefault == nil || *columnDefault != "''::text" {
		t.Fatalf("score_source column_default = %v, want ''::text", columnDefault)
	}

	// An explicit NULL must be rejected by the NOT NULL constraint, so the
	// library list scan can never hit "cannot scan NULL into *string" again.
	_, err = pool.Exec(ctx, `
		INSERT INTO library_items (id, name, media_type, score_source)
		VALUES (gen_random_uuid(), 'Null Source', 'book', NULL)
	`)
	if err == nil {
		t.Fatal("expected INSERT with NULL score_source to fail (NOT NULL constraint)")
	}
}

func ptrInt(v int) *int { return &v }

func ptrFloat(v float64) *float64 { return &v }
