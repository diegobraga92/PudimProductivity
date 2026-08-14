package library

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// CreateInput is the payload used to create a single item (manual or imported).
type CreateInput struct {
	Name        string
	MediaType   MediaType
	ReleaseYear *int
	Done        bool
	Notes       string
	Score       *float64
	ScoreSource string
}

// UpdateInput carries the editable fields of an existing item. nil values mean
// "unchanged"; ReleaseYear and Score are double pointers so callers can
// distinguish "absent" (nil) from "set to null" (a nil inner pointer).
type UpdateInput struct {
	Name        *string
	MediaType   *MediaType
	ReleaseYear **int
	Done        *bool
	Notes       *string
	Score       **float64
	ScoreSource *string
}

// ImportError describes a single skipped row during a bulk import.
type ImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportResult is the outcome of a bulk import.
type ImportResult struct {
	Imported int            `json:"imported"`
	Items    []ItemResponse `json:"items"`
	Errors   []ImportError  `json:"errors"`
}

// LibraryService coordinates the library module.
type LibraryService struct {
	repo  Repository
	audit audit.Logger
	bus   eventbus.Bus
}

func NewLibraryService(repo Repository, auditLogger audit.Logger, bus eventbus.Bus) *LibraryService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &LibraryService{repo: repo, audit: auditLogger, bus: bus}
}

// Create validates and persists a single item.
func (s *LibraryService) Create(ctx context.Context, in CreateInput) (*Item, error) {
	item, err := NewItem(shared.NewUUID(), in.Name, in.MediaType, in.ReleaseYear, in.Done, in.Notes, in.Score, in.ScoreSource)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	log.Info().Ctx(ctx).Str("item_id", item.ID).Str("media_type", string(item.MediaType)).Str("name", item.Name).Msg("library item added")
	s.audit.Log(ctx, audit.ActionLibraryItemAdded, audit.ResourceLibraryItems, item.ID, nil, map[string]any{
		"name": item.Name, "media_type": item.MediaType, "done": item.Done,
		"score": item.Score, "score_source": item.ScoreSource,
	})
	s.publish(ctx, eventbus.EventLibraryItemAdded, toResponse(item))
	return item, nil
}

// Import validates a batch of items and persists the valid ones in a single
// transaction. Invalid rows are skipped and reported; valid rows are inserted.
func (s *LibraryService) Import(ctx context.Context, in []CreateInput) (*ImportResult, error) {
	var items []*Item
	errs := make([]ImportError, 0)
	for i, input := range in {
		item, err := NewItem(shared.NewUUID(), input.Name, input.MediaType, input.ReleaseYear, input.Done, input.Notes, input.Score, input.ScoreSource)
		if err != nil {
			errs = append(errs, ImportError{Row: i + 1, Message: err.Error()})
			continue
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.repo.Import(ctx, items); err != nil {
			return nil, err
		}
	}

	if len(items) > 0 || len(errs) > 0 {
		summary := map[string]any{"imported": len(items), "skipped": len(errs)}
		s.audit.Log(ctx, audit.ActionLibraryItemsImported, audit.ResourceLibraryItems, "", nil, summary)
		s.publish(ctx, eventbus.EventLibraryItemsImported, summary)
	}

	return &ImportResult{Imported: len(items), Items: toResponses(items), Errors: errs}, nil
}

func (s *LibraryService) Get(ctx context.Context, id string) (*Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *LibraryService) List(ctx context.Context, mediaType string, done *bool) ([]*Item, error) {
	return s.repo.List(ctx, mediaType, done)
}

// Update merges the editable fields and persists the result, returning the
// updated item.
func (s *LibraryService) Update(ctx context.Context, id string, in UpdateInput) (*Item, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err // ErrNotFound passes through
	}

	if in.Name != nil {
		current.Name = *in.Name
	}
	if in.MediaType != nil {
		current.MediaType = *in.MediaType
	}
	if in.ReleaseYear != nil {
		current.ReleaseYear = *in.ReleaseYear
	}
	if in.Done != nil {
		current.Done = *in.Done
	}
	if in.Notes != nil {
		current.Notes = *in.Notes
	}
	if in.Score != nil {
		current.Score = *in.Score
	}
	if in.ScoreSource != nil {
		current.ScoreSource = *in.ScoreSource
	}

	// Re-validate the merged state.
	if err := validateItem(current); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}

	log.Info().Ctx(ctx).Str("item_id", id).Msg("library item updated")
	s.audit.Log(ctx, audit.ActionLibraryItemUpdated, audit.ResourceLibraryItems, id, nil, map[string]any{
		"name": current.Name, "media_type": current.MediaType, "done": current.Done,
		"score": current.Score, "score_source": current.ScoreSource,
	})
	s.publish(ctx, eventbus.EventLibraryItemUpdated, toResponse(current))
	return current, nil
}

func (s *LibraryService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err // ErrNotFound passes through
	}
	s.audit.Log(ctx, audit.ActionLibraryItemDeleted, audit.ResourceLibraryItems, id, nil, nil)
	s.publish(ctx, eventbus.EventLibraryItemDeleted, map[string]any{"id": id})
	return nil
}

// validateItem re-checks a merged item without re-generating an ID.
func validateItem(item *Item) error {
	if _, err := NewItem(item.ID, item.Name, item.MediaType, item.ReleaseYear, item.Done, item.Notes, item.Score, item.ScoreSource); err != nil {
		return fmt.Errorf("update library item: %w", err)
	}
	return nil
}

func (s *LibraryService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish library event")
	}
}
