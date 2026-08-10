package booktrack

import (
	"context"
	"errors"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/booktrack/googlebooks"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

// --- fakes ---

type fakeRepo struct {
	saved       *Book
	createErr   error
	updateErr   error
	deleteErr   error
	updatedTo   BookStatus
	deletedID   string
	createdISBN string
}

func (f *fakeRepo) Create(ctx context.Context, book *Book) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.createdISBN == book.ISBN {
		return ErrDuplicateISBN
	}
	f.saved = book
	f.createdISBN = book.ISBN
	return nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (*Book, error) {
	if f.saved == nil || f.saved.ID != id {
		return nil, ErrNotFound
	}
	return f.saved, nil
}
func (f *fakeRepo) GetByISBN(ctx context.Context, isbn string) (*Book, error) {
	if f.saved == nil || f.saved.ISBN != isbn {
		return nil, ErrNotFound
	}
	return f.saved, nil
}
func (f *fakeRepo) List(ctx context.Context, status string) ([]*Book, error) {
	if f.saved == nil {
		return nil, nil
	}
	return []*Book{f.saved}, nil
}
func (f *fakeRepo) UpdateStatus(ctx context.Context, id string, status BookStatus) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updatedTo = status
	return nil
}
func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedID = id
	return nil
}

type fakeLookup struct {
	info  *googlebooks.BookInfo
	err   error
	calls int
}

func (f *fakeLookup) LookupByISBN(ctx context.Context, isbn string) (*googlebooks.BookInfo, error) {
	f.calls++
	return f.info, f.err
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

// --- tests ---

func TestService_AddByISBN_PublishesEventAndAudits(t *testing.T) {
	repo := &fakeRepo{}
	lookup := &fakeLookup{info: &googlebooks.BookInfo{
		ISBN: "9781250237231", Title: "Permanent Record", Authors: []string{"Edward Snowden"},
	}}
	spyAudit := &auditSpy{}
	spyBus := &busSpy{}
	svc := NewBookService(repo, lookup, spyAudit, spyBus)

	book, err := svc.AddByISBN(context.Background(), "978-1250237231")
	if err != nil {
		t.Fatalf("AddByISBN: %v", err)
	}
	if book.ISBN != "9781250237231" {
		t.Fatalf("ISBN not normalized: %q", book.ISBN)
	}
	if book.Status != StatusWantToRead {
		t.Fatalf("status = %q, want want_to_read", book.Status)
	}
	if len(spyAudit.actions) != 1 || spyAudit.actions[0] != audit.ActionBookAdded {
		t.Fatalf("audit = %v", spyAudit.actions)
	}
	if len(spyBus.types) != 1 || spyBus.types[0] != eventbus.EventBookAdded {
		t.Fatalf("events = %v", spyBus.types)
	}
}

func TestService_AddByISBN_NoLookupConfigured(t *testing.T) {
	svc := NewBookService(&fakeRepo{}, nil, nil, nil)
	if _, err := svc.AddByISBN(context.Background(), "9781250237231"); err == nil {
		t.Fatal("expected error when lookup is not configured")
	}
}

func TestService_AddByISBN_DuplicateIsbn(t *testing.T) {
	repo := &fakeRepo{}
	lookup := &fakeLookup{info: &googlebooks.BookInfo{ISBN: "9781250237231", Title: "A"}}
	svc := NewBookService(repo, lookup, nil, nil)
	ctx := context.Background()

	if _, err := svc.AddByISBN(ctx, "9781250237231"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if _, err := svc.AddByISBN(ctx, "9781250237231"); !errors.Is(err, ErrDuplicateISBN) {
		t.Fatalf("want ErrDuplicateISBN, got %v", err)
	}
}

func TestService_AddManual_DefaultsStatus(t *testing.T) {
	svc := NewBookService(&fakeRepo{}, nil, nil, nil)
	book, err := svc.AddManual(context.Background(), AddInput{
		ISBN: "978-3-16-148410-0", Title: "Manual Book",
	})
	if err != nil {
		t.Fatalf("AddManual: %v", err)
	}
	if book.Status != StatusWantToRead {
		t.Fatalf("status = %q, want want_to_read", book.Status)
	}
	if book.ISBN != "9783161484100" {
		t.Fatalf("ISBN normalized to %q", book.ISBN)
	}
}

func TestService_AddManual_InvalidISBNRejected(t *testing.T) {
	svc := NewBookService(&fakeRepo{}, nil, nil, nil)
	if _, err := svc.AddManual(context.Background(), AddInput{Title: "No ISBN"}); err == nil {
		t.Fatal("expected error for empty ISBN")
	}
}

func TestService_UpdateStatusAndDeleteAudit(t *testing.T) {
	repo := &fakeRepo{}
	spyAudit := &auditSpy{}
	svc := NewBookService(repo, nil, spyAudit, nil)
	ctx := context.Background()

	if err := svc.UpdateStatus(ctx, "b1", StatusReading); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if err := svc.Delete(ctx, "b1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	want := []string{audit.ActionBookUpdated, audit.ActionBookDeleted}
	if len(spyAudit.actions) != 2 {
		t.Fatalf("audit = %v, want %v", spyAudit.actions, want)
	}
	for i := range want {
		if spyAudit.actions[i] != want[i] {
			t.Fatalf("audit = %v, want %v", spyAudit.actions, want)
		}
	}
}

func TestService_UpdateStatusInvalid(t *testing.T) {
	svc := NewBookService(&fakeRepo{}, nil, nil, nil)
	if err := svc.UpdateStatus(context.Background(), "b1", "archived"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}
