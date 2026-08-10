package recipe

import (
	"context"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupRecipeTestPostgres starts a fresh Postgres and applies all migrations.
func setupRecipeTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func TestRecipeRepository_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupRecipeTestPostgres(t)
	repo := NewPostgresRepository(pool)

	recipe, err := NewRecipe(
		"11111111-1111-1111-1111-111111111111", "Pancakes", "Fluffy", DifficultyEasy,
		10, 15, 4,
		ptr("https://img/x.jpg"), nil,
		[]string{"breakfast"},
		[]Ingredient{{ID: "22222222-2222-2222-2222-222222222222", Name: "Flour", Quantity: "2", Unit: "cups"}},
		[]Step{{ID: "33333333-3333-3333-3333-333333333333", StepNumber: 1, Instruction: "Mix"}},
	)
	if err != nil {
		t.Fatalf("NewRecipe: %v", err)
	}

	if err := repo.Create(ctx, recipe); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, recipe.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "Pancakes" || len(got.Tags) != 1 || len(got.Ingredients) != 1 || len(got.Steps) != 1 {
		t.Fatalf("GetByID returned incomplete recipe: %+v", got)
	}

	// Update with different children — old ones must be replaced, not merged.
	updated, err := NewRecipe(
		recipe.ID, "Pancakes v2", "Still fluffy", DifficultyMedium,
		15, 20, 6,
		nil, nil, []string{"breakfast", "quick"},
		[]Ingredient{{ID: "44444444-4444-4444-4444-444444444444", Name: "Egg", Quantity: "2"}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewRecipe updated: %v", err)
	}
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got2, err := repo.GetByID(ctx, recipe.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got2.Title != "Pancakes v2" || len(got2.Tags) != 2 || len(got2.Ingredients) != 1 || len(got2.Steps) != 0 {
		t.Fatalf("Update did not replace children: %+v", got2)
	}

	if err := repo.Delete(ctx, recipe.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, recipe.ID); err != ErrNotFound {
		t.Fatalf("GetByID after delete: want ErrNotFound, got %v", err)
	}
}

func TestRecipeRepository_ListFiltersAndPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupRecipeTestPostgres(t)
	repo := NewPostgresRepository(pool)

	recipes := []*Recipe{
		mustRecipe("55555555-0000-0000-0000-000000000001", "Tomato Soup", DifficultyEasy, []string{"soup", "vegan"}),
		mustRecipe("55555555-0000-0000-0000-000000000002", "Chicken Noodle Soup", DifficultyHard, []string{"soup", "meat"}),
		mustRecipe("55555555-0000-0000-0000-000000000003", "Caesar Salad", DifficultyEasy, []string{"salad"}),
	}
	for _, r := range recipes {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.Title, err)
		}
	}

	// Search filter.
	list, err := repo.List(ctx, ListFilter{Search: "soup"})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("search 'soup' = %d recipes, want 2", len(list))
	}

	// Tag filter.
	list, err = repo.List(ctx, ListFilter{Tags: []string{"vegan"}})
	if err != nil {
		t.Fatalf("List tags: %v", err)
	}
	if len(list) != 1 || list[0].ID != recipes[0].ID {
		t.Fatalf("tag filter = %d recipes, want the vegan soup", len(list))
	}

	// Difficulty filter.
	list, err = repo.List(ctx, ListFilter{Difficulty: "hard"})
	if err != nil {
		t.Fatalf("List difficulty: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Chicken Noodle Soup" {
		t.Fatalf("difficulty filter = %v", list)
	}

	// Pagination: limit 1 returns the newest (Caesar Salad was last created).
	page1, err := repo.List(ctx, ListFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("page size = %d, want 1", len(page1))
	}
	// Keyset cursor: page2 starts after page1's (created_at, id).
	page2, err := repo.List(ctx, ListFilter{Limit: 1, Cursor: &page1[0].CreatedAt, CursorID: page1[0].ID})
	if err != nil {
		t.Fatalf("List cursor: %v", err)
	}
	if len(page2) != 1 || page2[0].ID == page1[0].ID {
		t.Fatalf("page2 = %v, want a different recipe", page2)
	}
}

func mustRecipe(id, title string, diff Difficulty, tags []string) *Recipe {
	r, err := NewRecipe(id, title, "", diff, 5, 10, 2, nil, nil, tags, nil, nil)
	if err != nil {
		panic(err)
	}
	return r
}
