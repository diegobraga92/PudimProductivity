package mealplan

import (
	"context"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

type fakeRepo struct {
	batches  [][]*Item
	updated  *Item
	deleted  string
	clearIDs []string
	listable []*Item
}

func (f *fakeRepo) CreateBatch(_ context.Context, items []*Item) error {
	f.batches = append(f.batches, items)
	return nil
}

func (f *fakeRepo) List(_ context.Context) ([]*Item, error) { return f.listable, nil }

func (f *fakeRepo) Update(_ context.Context, item *Item) error {
	f.updated = item
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.deleted = id
	return nil
}

func (f *fakeRepo) DeleteAll(_ context.Context) ([]string, error) { return f.clearIDs, nil }

type auditSpy struct {
	actions []string
}

func (s *auditSpy) Log(_ context.Context, action, _, _ string, _, _ any) {
	s.actions = append(s.actions, action)
}

type busSpy struct {
	types []eventbus.EventType
}

func (b *busSpy) Publish(_ context.Context, typ eventbus.EventType, _ interface{}) error {
	b.types = append(b.types, typ)
	return nil
}

func (b *busSpy) Subscribe(_ context.Context, _ eventbus.Handler) (func(), error) {
	return func() {}, nil
}

func (b *busSpy) Close() error { return nil }

func TestCreateFansOutAcrossDays(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	service := NewService(repo, spyAudit, spyBus)

	items, err := service.Create(context.Background(), CreateInput{
		Days:     []Weekday{Monday, Tuesday, Friday, Monday},
		Meal:     MealLunch,
		Name:     "Rice",
		Amount:   "100g",
		Calories: 120,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3 after deduplicating days", len(items))
	}
	if len(repo.batches) != 1 || len(repo.batches[0]) != 3 {
		t.Fatalf("expected a single batch of 3, got %+v", repo.batches)
	}
	if len(spyBus.types) != 3 {
		t.Fatalf("events = %v, want 3", spyBus.types)
	}
	for _, typ := range spyBus.types {
		if typ != eventbus.EventMealPlanItemCreated {
			t.Fatalf("unexpected event type %q", typ)
		}
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionMealPlanItemCreated {
		t.Fatalf("audit actions = %v", spyAudit.actions)
	}
}

func TestCreateRequiresAtLeastOneDay(t *testing.T) {
	service := NewService(&fakeRepo{}, nil, nil)
	if _, err := service.Create(context.Background(), CreateInput{Meal: MealLunch, Name: "Rice"}); err == nil {
		t.Fatal("expected error when no day is selected")
	}
}

func TestCreatePropagatesValidationError(t *testing.T) {
	service := NewService(&fakeRepo{}, nil, nil)
	_, err := service.Create(context.Background(), CreateInput{Days: []Weekday{Monday}, Meal: MealLunch, Name: "  "})
	if err == nil {
		t.Fatal("expected error for blank name")
	}
}

func TestUpdateAndDelete(t *testing.T) {
	repo := &fakeRepo{}
	spyBus := &busSpy{}
	service := NewService(repo, nil, spyBus)
	ctx := context.Background()

	item, err := service.Update(ctx, "id-1", UpdateInput{
		Day: Saturday, Meal: MealDinner, Name: "Pasta", Calories: 500,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if repo.updated == nil || item.ID != "id-1" || item.Name != "Pasta" {
		t.Fatalf("repository Update not called with the item: %+v", repo.updated)
	}

	if err := service.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.deleted != "id-1" {
		t.Fatal("repository Delete not called")
	}

	want := []eventbus.EventType{eventbus.EventMealPlanItemUpdated, eventbus.EventMealPlanItemDeleted}
	if len(spyBus.types) != len(want) {
		t.Fatalf("events = %v, want %v", spyBus.types, want)
	}
	for i, typ := range want {
		if spyBus.types[i] != typ {
			t.Fatalf("events = %v, want %v", spyBus.types, want)
		}
	}
}

func TestListDelegatesToRepository(t *testing.T) {
	repo := &fakeRepo{listable: []*Item{{ID: "id-1"}}}
	service := NewService(repo, nil, nil)

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].ID != "id-1" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestClearRemovesEveryItem(t *testing.T) {
	repo := &fakeRepo{clearIDs: []string{"id-1", "id-2"}}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	service := NewService(repo, spyAudit, spyBus)

	if err := service.Clear(context.Background()); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionMealPlanItemsCleared {
		t.Fatalf("audit actions = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 2 {
		t.Fatalf("events = %v, want 2", spyBus.types)
	}
	for _, typ := range spyBus.types {
		if typ != eventbus.EventMealPlanItemDeleted {
			t.Fatalf("unexpected event type %q", typ)
		}
	}
}

func TestClearOnEmptyPlanIsSilent(t *testing.T) {
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	service := NewService(&fakeRepo{}, spyAudit, spyBus)

	if err := service.Clear(context.Background()); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(spyAudit.actions) != 0 || len(spyBus.types) != 0 {
		t.Fatalf("empty clear logged %v and published %v", spyAudit.actions, spyBus.types)
	}
}
