// Package eventbus provides the event-dispatch abstraction used across the backend.
package eventbus

import (
	"context"
	"errors"
	"time"
)

// EventType identifies the kind of domain event. Payload shapes are documented in events.json.
type EventType string

const (
	EventTaskCreated     EventType = "task.created"
	EventTaskUpdated     EventType = "task.updated"
	EventTaskDeleted     EventType = "task.deleted"
	EventTaskCompleted   EventType = "task.completed"
	EventTaskUncompleted EventType = "task.uncompleted"

	EventLibraryItemAdded     EventType = "library.item.added"
	EventLibraryItemUpdated   EventType = "library.item.updated"
	EventLibraryItemDeleted   EventType = "library.item.deleted"
	EventLibraryItemsImported EventType = "library.items.imported"

	EventRecipeCreated EventType = "recipe.created"
	EventRecipeUpdated EventType = "recipe.updated"
	EventRecipeDeleted EventType = "recipe.deleted"

	EventTaskListShared   EventType = "tasklist.shared"
	EventTaskListUnshared EventType = "tasklist.unshared"
	EventTaskMerged       EventType = "task.merged"

	EventPresenceOnline  EventType = "presence.online"
	EventPresenceOffline EventType = "presence.offline"
)

// Event is the wire envelope pushed to subscribers.
type Event struct {
	// ID uniquely identifies the event instance. The in-memory bus leaves it empty.
	ID        string      `json:"id,omitempty"`
	Type      EventType   `json:"type"`
	Seq       int64       `json:"seq"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload,omitempty"`

	TraceID string `json:"-"`
	SpanID  string `json:"-"`
}

// ErrBusClosed is returned by Publish/Subscribe after Close has been called.
var ErrBusClosed = errors.New("event bus closed")

// Handler receives a single event.
type Handler func(ctx context.Context, event Event) error

// Bus is the event-dispatch abstraction.
type Bus interface {
	// Publish delivers an event to all subscribers in publish order.
	Publish(ctx context.Context, typ EventType, payload interface{}) error
	// Subscribe registers a handler. The returned function removes the handler
	// and is safe to call multiple times.
	Subscribe(ctx context.Context, handler Handler) (unsubscribe func(), err error)
	// Close stops the bus. Publish/Subscribe fail afterwards.
	Close() error
}
