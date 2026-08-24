package library

import (
	"context"
	"errors"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

type fakeRepo struct {
	saved     *Item
	imported  []*Item
	updated   *Item
	deleted   string
	createErr error
	updateErr error
	deleteErr error
	importErr error
}

func (f *fakeRepo) Create(ctx context.Context, item *Item) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.saved = item
	return nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (*Item, error) {
	if f.saved == nil || f.saved.ID != id {
		return nil, ErrNotFound
	}
	return f.saved, nil
}
func (f *fakeRepo) List(ctx context.Context, filter ListFilter) ([]*Item, error) {
	if f.saved == nil {
		return nil, nil
	}
	return []*Item{f.saved}, nil
}

func (f *fakeRepo) DistinctSubtypes(ctx context.Context, mediaType string) ([]string, error) {
	if f.saved == nil || f.saved.Subtype == "" {
		return []string{}, nil
	}
	if mediaType != "" && string(f.saved.MediaType) != mediaType {
		return []string{}, nil
	}
	return []string{f.saved.Subtype}, nil
}
func (f *fakeRepo) Update(ctx context.Context, item *Item) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = item
	f.saved = item
	return nil
}
func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = id
	return nil
}
func (f *fakeRepo) Import(ctx context.Context, items []*Item) error {
	if f.importErr != nil {
		return f.importErr
	}
	f.imported = items
	return nil
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
func (b *busSpy) Subscribe(_ context.Context, _ eventbus.Handler) (func(), error) {
	return func() {}, nil
}
func (b *busSpy) Close() error { return nil }

func ptrInt(v int) *int { return &v }

func ptrFloat(v float64) *float64 { return &v }

func TestService_Create_PublishesEventAndAudits(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewLibraryService(repo, spyAudit, spyBus)

	item, err := svc.Create(context.Background(), CreateInput{
		Name: "The Matrix", MediaType: MediaTypeMovie, ReleaseYear: ptrInt(1999), Done: false, Notes: "Sci-fi classic",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if item.Name != "The Matrix" || item.MediaType != MediaTypeMovie || item.Done {
		t.Fatalf("unexpected item: %+v", item)
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionLibraryItemAdded {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventLibraryItemAdded {
		t.Fatalf("events = %v", spyBus.types)
	}
}

func TestService_Create_Validation(t *testing.T) {
	svc := NewLibraryService(&fakeRepo{}, nil, nil)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateInput{Name: "", MediaType: MediaTypeBook}); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := svc.Create(ctx, CreateInput{Name: "X", MediaType: "vinyl"}); err == nil {
		t.Fatal("expected error for invalid media type")
	}
	if _, err := svc.Create(ctx, CreateInput{Name: "X", MediaType: MediaTypeGame, ReleaseYear: ptrInt(1400)}); err == nil {
		t.Fatal("expected error for out-of-range year")
	}
	if _, err := svc.Create(ctx, CreateInput{Name: "X", MediaType: MediaTypeMovie, Score: ptrFloat(150)}); err == nil {
		t.Fatal("expected error for out-of-range score")
	}
	if _, err := svc.Create(ctx, CreateInput{Name: "X", MediaType: MediaTypeMovie, ScoreSource: "imdb"}); err == nil {
		t.Fatal("expected error for score source without a score")
	}
}

func TestService_Update_TogglesDone(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	svc := NewLibraryService(repo, spyAudit, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, CreateInput{Name: "Inception", MediaType: MediaTypeMovie})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	done := true
	updated, err := svc.Update(ctx, item.ID, UpdateInput{Done: &done})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Done {
		t.Fatal("expected done=true after update")
	}
	if len(spyAudit.actions) != 2 || spyAudit.actions[1] != audit.ActionLibraryItemUpdated {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
}

func TestService_Update_ClearsReleaseYear(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewLibraryService(repo, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, CreateInput{Name: "Portal", MediaType: MediaTypeGame, ReleaseYear: ptrInt(2007)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var nilYear *int
	updated, err := svc.Update(ctx, item.ID, UpdateInput{ReleaseYear: &nilYear})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ReleaseYear != nil {
		t.Fatalf("expected release_year cleared, got %v", *updated.ReleaseYear)
	}
}

func TestService_Update_SetsAndClearsScore(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewLibraryService(repo, nil, nil)
	ctx := context.Background()

	item, err := svc.Create(ctx, CreateInput{Name: "The Matrix", MediaType: MediaTypeMovie})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Set a score + source.
	score := 8.7
	scorePtr := &score
	source := "imdb"
	updated, err := svc.Update(ctx, item.ID, UpdateInput{Score: &scorePtr, ScoreSource: &source})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Score == nil || *updated.Score != 8.7 || updated.ScoreSource != "imdb" {
		t.Fatalf("expected score 8.7/imdb after update, got %+v", updated)
	}

	// Clear the score and its source.
	var nilScore *float64
	emptySource := ""
	updated, err = svc.Update(ctx, item.ID, UpdateInput{Score: &nilScore, ScoreSource: &emptySource})
	if err != nil {
		t.Fatalf("Update (clear): %v", err)
	}
	if updated.Score != nil || updated.ScoreSource != "" {
		t.Fatalf("expected score cleared, got %+v", updated)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := NewLibraryService(&fakeRepo{}, nil, nil)
	if _, err := svc.Update(context.Background(), "missing", UpdateInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestService_Import_SkipsInvalidRows(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewLibraryService(repo, spyAudit, spyBus)

	result, err := svc.Import(context.Background(), []CreateInput{
		{Name: "Good Movie", MediaType: MediaTypeMovie},
		{Name: "", MediaType: MediaTypeMovie},  // empty name → skipped
		{Name: "Bad Type", MediaType: "vinyl"}, // invalid type → skipped
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 1 || len(result.Items) != 1 {
		t.Fatalf("imported = %d, want 1", result.Imported)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("errors = %d, want 2", len(result.Errors))
	}
	if len(repo.imported) != 1 {
		t.Fatalf("repo received %d items, want 1", len(repo.imported))
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionLibraryItemsImported {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventLibraryItemsImported {
		t.Fatalf("events = %v", spyBus.types)
	}
}

func TestService_Delete_Audits(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	svc := NewLibraryService(repo, spyAudit, nil)
	ctx := context.Background()

	if err := svc.Delete(ctx, "item-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.deleted != "item-1" {
		t.Fatalf("deleted = %q", repo.deleted)
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionLibraryItemDeleted {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
}
