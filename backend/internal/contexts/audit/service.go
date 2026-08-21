package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

// TODO: Add a proper shutdown

const (
	auditWriteTimeout        = 5 * time.Second
	maxAuditQueryLimit       = 100
	defaultAuditQueryLimit   = 50
	defaultAuditLookbackDays = 7
	defaultAuditRecentDays   = 1
)

// Service provides the core audit logging functionality.
type Service struct {
	repo   Repository
	events chan *Entry
}

// NewService creates a new audit Service instance.
func NewService(repo Repository, bufferSize int) *Service {
	s := &Service{
		repo:   repo,
		events: make(chan *Entry, bufferSize),
	}

	go s.worker()

	return s
}

// Log records an audit entry async.
// Data is retrieve from context and comparison between old and new.
// If the internal buffer is full, the entry is dropped and a warning is logged.
func (s *Service) Log(ctx context.Context, action, resource, resourceID string, oldValues, newValues any) {
	actorID := httpx.GetUserID(ctx)
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
		ID:         uuid.NewUUID(),
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
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		if err := s.repo.Insert(ctx, entry); err != nil {
			log.Error().Err(err).
				Str("action", entry.Action).
				Str("resource", entry.Resource).
				Msg("failed to persist audit log entry")
		}
		cancel()
	}
}

// Query retrieves audit entries based on the provided options.
func (s *Service) Query(ctx context.Context, opts QueryOptions) ([]Entry, error) {
	limit := opts.Limit
	if limit <= 0 || limit > maxAuditQueryLimit {
		limit = defaultAuditQueryLimit
	}
	offset := max(opts.Offset, 0)

	switch {
	case opts.Resource != "" && opts.ResourceID != "":
		return s.repo.ListByResource(ctx, opts.Resource, opts.ResourceID, limit, offset)
	case opts.ActorID != "":
		return s.repo.ListByActor(ctx, opts.ActorID, limit, offset)
	case opts.Action != "":
		since := opts.Since
		if since.IsZero() {
			since = time.Now().AddDate(0, 0, -defaultAuditLookbackDays)
		}
		return s.repo.ListByAction(ctx, opts.Action, since, limit, offset)
	default:
		since := opts.Since
		if since.IsZero() {
			since = time.Now().AddDate(0, 0, -defaultAuditRecentDays)
		}
		return s.repo.ListByAction(ctx, "", since, limit, offset)
	}
}

// QueryOptions defines the filters and pagination parameters for retrieving audit logs.
type QueryOptions struct {
	Resource   string
	ResourceID string
	ActorID    string
	Action     string
	Since      time.Time
	Limit      int
	Offset     int
}
