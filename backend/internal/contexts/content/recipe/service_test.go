package recipe

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/media"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

type fakeRepo struct {
	createCalled bool
	updateCalled bool
	deleteCalled bool
	saved        *Recipe
}

func (f *fakeRepo) Create(ctx context.Context, recipe *Recipe) error {
	f.createCalled = true
	f.saved = recipe
	return nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (*Recipe, error) {
	if f.saved != nil && f.saved.ID == id {
		return f.saved, nil
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) List(ctx context.Context, filter ListFilter) ([]*Recipe, error) {
	if f.saved == nil {
		return nil, nil
	}
	return []*Recipe{f.saved}, nil
}
func (f *fakeRepo) Update(ctx context.Context, recipe *Recipe) error {
	f.updateCalled = true
	f.saved = recipe
	return nil
}
func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	f.deleteCalled = true
	f.saved = nil
	return nil
}

type auditSpy struct {
	actions []string
}

func (s *auditSpy) Log(_ context.Context, action, _, _ string, _, _ any) {
	s.actions = append(s.actions, action)
}

type busSpy struct {
	types []eventbus.EventType
}

func (b *busSpy) Publish(_ context.Context, typ eventbus.EventType, _ any) error {
	b.types = append(b.types, typ)
	return nil
}
func (b *busSpy) Subscribe(_ context.Context, _ eventbus.Handler) (func(), error) {
	return func() {}, nil
}
func (b *busSpy) Close() error { return nil }

func TestService_CreatePublishesEventAndAudits(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewService(repo, spyAudit, spyBus, nil)

	in := CreateInput{
		Title: "Pasta", Difficulty: DifficultyEasy, Servings: 2,
		Tags:        []string{"Italian"},
		Ingredients: []Ingredient{{Name: "Pasta"}},
		Steps:       []Step{{Instruction: "Boil"}},
	}
	recipe, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !repo.createCalled {
		t.Fatal("repository Create not called")
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionRecipeCreated {
		t.Fatalf("audit actions = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventRecipeCreated {
		t.Fatalf("events = %v", spyBus.types)
	}
	// Child IDs + sort orders assigned by the service.
	if recipe.Ingredients[0].ID == "" {
		t.Fatal("ingredient ID not assigned")
	}
	if recipe.Steps[0].StepNumber != 1 {
		t.Fatalf("step number = %d, want 1", recipe.Steps[0].StepNumber)
	}
}

func TestService_CreateValidationErrorDoesNotPersist(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil, nil, nil)

	_, err := svc.Create(context.Background(), CreateInput{Title: "", Difficulty: DifficultyEasy, Servings: 1})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
	if repo.createCalled {
		t.Fatal("repository Create should not run on validation error")
	}
}

func TestService_UpdateAndDeleteFireEvents(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewService(repo, spyAudit, spyBus, nil)

	ctx := context.Background()
	_, err := svc.Update(ctx, "id-1", CreateInput{
		Title: "Updated", Difficulty: DifficultyHard, Servings: 4,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !repo.updateCalled {
		t.Fatal("repository Update not called")
	}
	if err := svc.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !repo.deleteCalled {
		t.Fatal("repository Delete not called")
	}

	wantEvents := []eventbus.EventType{eventbus.EventRecipeUpdated, eventbus.EventRecipeDeleted}
	if len(spyBus.types) != 2 {
		t.Fatalf("events = %v, want %v", spyBus.types, wantEvents)
	}
	for i := range wantEvents {
		if spyBus.types[i] != wantEvents[i] {
			t.Fatalf("events = %v, want %v", spyBus.types, wantEvents)
		}
	}
}

func TestService_GenerateUploadURLWithoutMedia(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil, nil, nil)
	_, err := svc.GenerateUploadURL(context.Background(), "id-1", "image/jpeg", "a.jpg")
	if err == nil {
		t.Fatal("expected error when media is not configured")
	}
}

func TestService_GenerateUploadURLUnknownRecipe(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil, nil, &fakeUploader{})
	_, err := svc.GenerateUploadURL(context.Background(), "missing", "image/jpeg", "a.jpg")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

type fakeUploader struct{}

func (f *fakeUploader) GenerateUploadURL(ctx context.Context, key, contentType string, ttl time.Duration) (*media.UploadURL, error) {
	return &media.UploadURL{URL: "https://presigned", Key: key}, nil
}
