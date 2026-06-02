package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

type Service struct {
	repo   Repository
	events chan *Entry
}

func NewService(repo Repository, bufferSize int) *Service {
	// bufferSize controls the maximum number of queued audit entries before blocking.
	s := &Service{
		repo:   repo,
		events: make(chan *Entry, bufferSize),
	}

	go s.worker()

	return s
}

func (s *Service) Log(ctx context.Context, action, resource, resourceID string, oldValues, newValues interface{}) {
	actorID := shared.GetUserID(ctx)
	if actorID == "" {
		actorID = "system"
	}

	var oldJSON, newJSON json.RawMessage
	if oldValues != nil {
		data, err := json.Marshal(oldValues)
		if err != nil {
			log.Warn().Err(err).Str("action", action).Msg("failed to marshal old values for audit")
			oldJSON = json.RawMessage(`{}`)
		} else {
			oldJSON = data
		}
	}
	if newValues != nil {
		data, err := json.Marshal(newValues)
		if err != nil {
			log.Warn().Err(err).Str("action", action).Msg("failed to marshal new values for audit")
			newJSON = json.RawMessage(`{}`)
		} else {
			newJSON = data
		}
	}

	entry := &Entry{
		ID:         shared.NewUUID(),
		ActorID:    actorID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		OldValues:  oldJSON,
		NewValues:  newJSON,
		CreatedAt:  time.Now().UTC(),
	}

	// Non-blocking send to channel; drop if buffer full
	// Returns immediately; the actual DB write happens in the worker goroutine
	select {
	case s.events <- entry:
	default:
		log.Warn().
			Str("action", action).
			Str("resource", resource).
			Msg("audit log buffer full — dropping entry")
	}
}

func (s *Service) worker() {
	for entry := range s.events {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.repo.Insert(ctx, entry); err != nil {
			log.Error().Err(err).
				Str("action", entry.Action).
				Str("resource", entry.Resource).
				Msg("failed to persist audit log entry")
		}
		cancel()
	}
}

func (s *Service) Query(ctx context.Context, opts QueryOptions) ([]Entry, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	switch {
	case opts.Resource != "" && opts.ResourceID != "":
		return s.repo.ListByResource(ctx, opts.Resource, opts.ResourceID, limit, offset)
	case opts.ActorID != "":
		return s.repo.ListByActor(ctx, opts.ActorID, limit, offset)
	case opts.Action != "":
		since := opts.Since
		if since.IsZero() {
			since = time.Now().AddDate(0, 0, -7) // default: last 7 days
		}
		return s.repo.ListByAction(ctx, opts.Action, since, limit, offset)
	default:
		since := opts.Since
		if since.IsZero() {
			since = time.Now().AddDate(0, 0, -1) // default: last 24 hours
		}
		return s.repo.ListByAction(ctx, "", since, limit, offset)
	}
}

type QueryOptions struct {
	Resource   string
	ResourceID string
	ActorID    string
	Action     string
	Since      time.Time
	Limit      int
	Offset     int
}