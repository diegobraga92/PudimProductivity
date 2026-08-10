package mealplan

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/recipe"
)

// --- fakes ---

type fakeRepo struct {
	saved         *MealPlan
	savedShopping []ShoppingItem
	list          []*MealPlan
}

func (f *fakeRepo) Create(ctx context.Context, plan *MealPlan) error {
	f.saved = plan
	return nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (*MealPlan, error) {
	if f.saved != nil && f.saved.ID == id {
		return f.saved, nil
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) List(ctx context.Context) ([]*MealPlan, error) {
	if f.list != nil {
		return f.list, nil
	}
	if f.saved != nil {
		return []*MealPlan{f.saved}, nil
	}
	return nil, nil
}
func (f *fakeRepo) Update(ctx context.Context, plan *MealPlan) error {
	f.saved = plan
	return nil
}
func (f *fakeRepo) Delete(ctx context.Context, id string) error { return nil }
func (f *fakeRepo) UpdateSlot(ctx context.Context, planID, slotID string, recipeID *string, notes *string) error {
	return nil
}
func (f *fakeRepo) ReplaceShoppingList(ctx context.Context, planID string, items []ShoppingItem) error {
	f.savedShopping = items
	return nil
}
func (f *fakeRepo) GetShoppingList(ctx context.Context, planID string) ([]ShoppingItem, error) {
	return f.savedShopping, nil
}
func (f *fakeRepo) ToggleShoppingItem(ctx context.Context, planID, itemID string) error { return nil }
func (f *fakeRepo) SetPublished(ctx context.Context, id string) error                  { return nil }

type fakeRecipeReader struct{ recipes map[string]*recipe.Recipe }

func (f fakeRecipeReader) Get(ctx context.Context, id string) (*recipe.Recipe, error) {
	r, ok := f.recipes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return r, nil
}

type auditSpy struct{ actions []string }

func (s *auditSpy) Log(_ context.Context, action, _, _ string, _, _ any) {
	s.actions = append(s.actions, action)
}

type busSpy struct{ types []eventbus.EventType }

func (b *busSpy) Publish(_ context.Context, typ eventbus.EventType, _ any) error {
	b.types = append(b.types, typ)
	return nil
}
func (b *busSpy) Subscribe(_ context.Context, _ eventbus.Handler) (func(), error) { return func() {}, nil }
func (b *busSpy) Close() error                                                   { return nil }

// --- tests ---

func TestService_CreateFiresEventAndAudit(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewMealPlanService(repo, nil, spyAudit, spyBus)

	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	plan, err := svc.Create(context.Background(), CreateInput{Name: "Week 1", StartDate: start, EndDate: end})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionMealPlanCreated {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventMealPlanCreated {
		t.Fatalf("events = %v", spyBus.types)
	}
	_ = plan
}

func TestService_PublishEmitsEvent(t *testing.T) {
	spyBus := &busSpy{}
	svc := NewMealPlanService(&fakeRepo{}, nil, nil, spyBus)
	if err := svc.Publish(context.Background(), "plan-1"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventMealPlanPublished {
		t.Fatalf("events = %v", spyBus.types)
	}
}

func TestService_GenerateShoppingListAggregates(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	r1 := "r1"
	r2 := "r2"

	repo := &fakeRepo{saved: &MealPlan{
		ID: "plan-1", StartDate: start, EndDate: end,
		Slots: []MealSlot{
			{Date: start, MealType: MealDinner, RecipeID: &r1},
			{Date: start.AddDate(0, 0, 1), MealType: MealLunch, RecipeID: &r2},
			{Date: start.AddDate(0, 0, 1), MealType: MealDinner, RecipeID: &r1}, // same recipe reused
		},
	}}

	reader := fakeRecipeReader{recipes: map[string]*recipe.Recipe{
		r1: mustRecipeWithIngredients("r1", []recipe.Ingredient{
			{Name: "Flour", Quantity: "2", Unit: "cups"},
			{Name: "Egg", Quantity: "2"},
		}),
		r2: mustRecipeWithIngredients("r2", []recipe.Ingredient{
			{Name: "Flour", Quantity: "1", Unit: "cups"},
			{Name: "Tomato", Quantity: "3"},
		}),
	}}

	svc := NewMealPlanService(repo, reader, nil, nil)
	items, err := svc.GenerateShoppingList(context.Background(), "plan-1")
	if err != nil {
		t.Fatalf("GenerateShoppingList: %v", err)
	}

	byName := map[string]string{}
	for _, item := range items {
		byName[item.IngredientName] = item.QuantityAgg
	}
	// Same recipe used twice → its ingredients counted once; flour from both
	// recipes is merged: 2 + 1 = 3 cups.
	if byName["flour"] != "3" {
		t.Fatalf("flour qty = %q, want 3 (items: %+v)", byName["flour"], items)
	}
	if byName["egg"] != "2" {
		t.Fatalf("egg qty = %q, want 2", byName["egg"])
	}
	if byName["tomato"] != "3" {
		t.Fatalf("tomato qty = %q, want 3", byName["tomato"])
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 aggregated items, got %+v", items)
	}
}

func TestService_GenerateShoppingListNoRecipesModule(t *testing.T) {
	svc := NewMealPlanService(&fakeRepo{}, nil, nil, nil)
	if _, err := svc.GenerateShoppingList(context.Background(), "plan-1"); err == nil {
		t.Fatal("expected error when recipes module is not configured")
	}
}

func mustRecipeWithIngredients(id string, ingredients []recipe.Ingredient) *recipe.Recipe {
	r, err := recipe.NewRecipe(id, "Test", "", recipe.DifficultyEasy, 5, 5, 2, nil, nil, nil, ingredients, nil)
	if err != nil {
		panic(err)
	}
	return r
}
