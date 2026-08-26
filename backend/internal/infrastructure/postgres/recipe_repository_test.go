package postgres_test

import (
	"testing"

	recipedomain "github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/recipe"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

func TestRecipeRepository_CRUD(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	repo := postgres.NewRecipeRepository(pool)

	recipe, err := recipedomain.NewRecipe(
		"11111111-1111-1111-1111-111111111111", "Pancakes", "Fluffy", recipedomain.DifficultyEasy,
		10, 15, 4,
		ptr("https://img/x.jpg"), nil,
		[]string{"breakfast"},
		[]recipedomain.Ingredient{{ID: "22222222-2222-2222-2222-222222222222", Name: "Flour", Quantity: "2", Unit: "cups"}},
		[]recipedomain.Step{{ID: "33333333-3333-3333-3333-333333333333", StepNumber: 1, Instruction: "Mix"}},
	)
	if err != nil {
		t.Fatalf("recipedomain.NewRecipe: %v", err)
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

	// Update with different children, old ones must be replaced, not merged.
	updated, err := recipedomain.NewRecipe(
		recipe.ID, "Pancakes v2", "Still fluffy", recipedomain.DifficultyMedium,
		15, 20, 6,
		nil, nil, []string{"breakfast", "quick"},
		[]recipedomain.Ingredient{{ID: "44444444-4444-4444-4444-444444444444", Name: "Egg", Quantity: "2"}},
		nil,
	)
	if err != nil {
		t.Fatalf("recipedomain.NewRecipe updated: %v", err)
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
	if _, err := repo.GetByID(ctx, recipe.ID); err != recipedomain.ErrNotFound {
		t.Fatalf("GetByID after delete: want recipedomain.ErrNotFound, got %v", err)
	}
}

func TestRecipeRepository_ListFiltersAndPagination(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	repo := postgres.NewRecipeRepository(pool)

	recipes := []*recipedomain.Recipe{
		mustRecipe("55555555-0000-0000-0000-000000000001", "Tomato Soup", recipedomain.DifficultyEasy, []string{"soup", "vegan"}),
		mustRecipe("55555555-0000-0000-0000-000000000002", "Chicken Noodle Soup", recipedomain.DifficultyHard, []string{"soup", "meat"}),
		mustRecipe("55555555-0000-0000-0000-000000000003", "Caesar Salad", recipedomain.DifficultyEasy, []string{"salad"}),
	}
	for _, r := range recipes {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.Title, err)
		}
	}

	// Search filter.
	list, err := repo.List(ctx, recipedomain.ListFilter{Search: "soup"})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("search 'soup' = %d recipes, want 2", len(list))
	}

	// Tag filter.
	list, err = repo.List(ctx, recipedomain.ListFilter{Tags: []string{"vegan"}})
	if err != nil {
		t.Fatalf("List tags: %v", err)
	}
	if len(list) != 1 || list[0].ID != recipes[0].ID {
		t.Fatalf("tag filter = %d recipes, want the vegan soup", len(list))
	}

	// Difficulty filter.
	list, err = repo.List(ctx, recipedomain.ListFilter{Difficulty: "hard"})
	if err != nil {
		t.Fatalf("List difficulty: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Chicken Noodle Soup" {
		t.Fatalf("difficulty filter = %v", list)
	}

	// Pagination: limit 1 returns the newest (Caesar Salad was last created).
	page1, err := repo.List(ctx, recipedomain.ListFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("page size = %d, want 1", len(page1))
	}
	// Keyset cursor: page2 starts after page1's (created_at, id).
	page2, err := repo.List(ctx, recipedomain.ListFilter{Limit: 1, Cursor: &page1[0].CreatedAt, CursorID: page1[0].ID})
	if err != nil {
		t.Fatalf("List cursor: %v", err)
	}
	if len(page2) != 1 || page2[0].ID == page1[0].ID {
		t.Fatalf("page2 = %v, want a different recipe", page2)
	}
}

func mustRecipe(id, title string, diff recipedomain.Difficulty, tags []string) *recipedomain.Recipe {
	r, err := recipedomain.NewRecipe(id, title, "", diff, 5, 10, 2, nil, nil, tags, nil, nil)
	if err != nil {
		panic(err)
	}
	return r
}

func ptr(s string) *string { return &s }
