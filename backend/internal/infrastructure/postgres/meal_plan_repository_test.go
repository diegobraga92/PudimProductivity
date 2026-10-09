package postgres_test

import (
	"testing"

	recipedomain "github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/recipe"
	mealplandomain "github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/mealplan"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

func TestMealPlanRepository_BatchListUpdateDelete(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	repo := postgres.NewMealPlanRepository(pool)

	items := []*mealplandomain.Item{
		mustItem("aaaaaaaa-0000-0000-0000-000000000001", mealplandomain.Monday, mealplandomain.MealBreakfast, "Oats", 300),
		mustItem("aaaaaaaa-0000-0000-0000-000000000002", mealplandomain.Tuesday, mealplandomain.MealBreakfast, "Oats", 300),
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	// The insert returns the generated timestamps so the HTTP response carries
	// the real values instead of the zero time.
	if items[0].CreatedAt.IsZero() || items[0].UpdatedAt.IsZero() {
		t.Fatalf("CreateBatch did not fill timestamps: %+v", items[0])
	}

	listed, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("List = %d items, want 2", len(listed))
	}

	updated := mustItem(items[0].ID, mealplandomain.Sunday, mealplandomain.MealDinner, "Pasta", 650)
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatal("Update did not fill the updated timestamp")
	}
	if err := repo.Update(ctx, mustItem("99999999-0000-0000-0000-000000000009", mealplandomain.Monday, mealplandomain.MealSnack, "Ghost", 1)); err != mealplandomain.ErrNotFound {
		t.Fatalf("Update on a missing item: want ErrNotFound, got %v", err)
	}

	listed, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List after update: %v", err)
	}
	var found *mealplandomain.Item
	for _, item := range listed {
		if item.ID == updated.ID {
			found = item
		}
	}
	if found == nil {
		t.Fatal("updated item missing from List")
	}
	if found.Name != "Pasta" || found.Day != mealplandomain.Sunday || found.Meal != mealplandomain.MealDinner || found.Calories != 650 {
		t.Fatalf("update not persisted: %+v", found)
	}

	if err := repo.Delete(ctx, updated.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := repo.Delete(ctx, updated.ID); err != mealplandomain.ErrNotFound {
		t.Fatalf("second Delete: want ErrNotFound, got %v", err)
	}
}

func TestMealPlanRepository_RecipeLinkClearedWhenRecipeDeleted(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	recipeRepo := postgres.NewRecipeRepository(pool)
	repo := postgres.NewMealPlanRepository(pool)

	recipe, err := recipedomain.NewRecipe(
		"cccccccc-0000-0000-0000-000000000001", "Pancakes", "", recipedomain.DifficultyEasy,
		10, 15, 4, nil, nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("NewRecipe: %v", err)
	}
	if err := recipeRepo.Create(ctx, recipe); err != nil {
		t.Fatalf("Create recipe: %v", err)
	}

	item := mustItem("aaaaaaaa-0000-0000-0000-000000000003", mealplandomain.Wednesday, mealplandomain.MealLunch, "Pancakes", 400)
	item.RecipeID = &recipe.ID
	if err := repo.CreateBatch(ctx, []*mealplandomain.Item{item}); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	if err := recipeRepo.Delete(ctx, recipe.ID); err != nil {
		t.Fatalf("Delete recipe: %v", err)
	}

	listed, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, got := range listed {
		if got.ID == item.ID && got.RecipeID != nil {
			t.Fatalf("recipe link not cleared on recipe delete: %+v", got)
		}
	}
}

func mustItem(id string, day mealplandomain.Weekday, meal mealplandomain.Meal, name string, calories int) *mealplandomain.Item {
	item, err := mealplandomain.NewItem(id, day, meal, name, "", calories, nil)
	if err != nil {
		panic(err)
	}
	return item
}
