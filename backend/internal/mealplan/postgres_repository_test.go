package mealplan

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func setupMealPlanTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
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
	pool, err := db.ConnectPool(ctx, shared.DatabaseConfig{URL: connStr, MaxConns: 5, MinConns: 1})
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

func TestMealPlanRepository_CRUDAndShoppingList(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupMealPlanTestPostgres(t)
	repo := NewPostgresRepository(pool)

	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

	plan, err := NewMealPlan("cccccccc-cccc-cccc-cccc-cccccccccccc", "Week 1", start, end, false, []MealSlot{
		{ID: "dddddddd-dddd-dddd-dddd-dddddddddddd", Date: start, MealType: MealDinner},
	})
	if err != nil {
		t.Fatalf("NewMealPlan: %v", err)
	}
	if err := repo.Create(ctx, plan); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Slots) != 1 || got.Slots[0].MealType != MealDinner {
		t.Fatalf("GetByID slots = %+v", got.Slots)
	}

	// Shopping list replace + read + toggle.
	items := []ShoppingItem{
		{ID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", IngredientName: "flour", QuantityAgg: "3", Unit: "cups"},
		{ID: "ffffffff-ffff-ffff-ffff-ffffffffffff", IngredientName: "egg", QuantityAgg: "2"},
	}
	if err := repo.ReplaceShoppingList(ctx, plan.ID, items); err != nil {
		t.Fatalf("ReplaceShoppingList: %v", err)
	}
	list, err := repo.GetShoppingList(ctx, plan.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("GetShoppingList: %v (len=%d)", err, len(list))
	}
	if err := repo.ToggleShoppingItem(ctx, plan.ID, items[0].ID); err != nil {
		t.Fatalf("ToggleShoppingItem: %v", err)
	}
	list, _ = repo.GetShoppingList(ctx, plan.ID)
	found := false
	for _, item := range list {
		if item.IngredientName == "flour" && !item.IsChecked {
			t.Fatalf("flour should be checked after toggle: %+v", list)
		}
		if item.IngredientName == "flour" {
			found = true
		}
	}
	if !found {
		t.Fatalf("flour missing from list: %+v", list)
	}

	// Publish + delete (CASCADE clears slots + shopping list).
	if err := repo.SetPublished(ctx, plan.ID); err != nil {
		t.Fatalf("SetPublished: %v", err)
	}
	got, _ = repo.GetByID(ctx, plan.ID)
	if !got.IsPublished {
		t.Fatal("plan should be published")
	}
	if err := repo.Delete(ctx, plan.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ = repo.GetShoppingList(ctx, plan.ID)
	if len(list) != 0 {
		t.Fatalf("shopping list should cascade-delete, got %d items", len(list))
	}
}
