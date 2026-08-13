// Package eventbus provides the event-dispatch abstraction used across the
// backend. Phase 2 introduces an in-memory implementation; Phase 3 will add a
// RabbitMQ-backed implementation that satisfies the same Bus interface so that
// producers (task service) and consumers (sync hub) do not change.
package eventbus

import (
	"context"
	"errors"
	"time"
)

// EventType identifies the kind of domain event. Payload shapes are documented
// in api/ws/events-v1.json.
type EventType string

const (
	// EventTaskCreated is published after a task is persisted.
	EventTaskCreated EventType = "task.created"
	// EventTaskUpdated is published after a task mutation is persisted.
	EventTaskUpdated EventType = "task.updated"
	// EventTaskDeleted is published after a task is removed.
	EventTaskDeleted EventType = "task.deleted"
	// EventTaskCompleted is published after a habit task is completed for a date.
	EventTaskCompleted EventType = "task.completed"
	// EventTaskUncompleted is published after a habit task completion is removed.
	EventTaskUncompleted EventType = "task.uncompleted"

	// Library media tracking (replaces Phase 5 book tracking).
	EventLibraryItemAdded     EventType = "library.item.added"
	EventLibraryItemUpdated   EventType = "library.item.updated"
	EventLibraryItemDeleted   EventType = "library.item.deleted"
	EventLibraryItemsImported EventType = "library.items.imported"

	// Phase 5: meal planning.
	EventMealPlanCreated   EventType = "mealplan.created"
	EventMealPlanPublished EventType = "mealplan.published"

	// Phase 5a: recipes.
	EventRecipeCreated EventType = "recipe.created"
	EventRecipeUpdated EventType = "recipe.updated"
	EventRecipeDeleted EventType = "recipe.deleted"

	// Phase 8: collaboration & multi-user.
	// EventTaskListShared is published when a user shares a task list with
	// another user. Payload: {list_id, shared_with, role, shared_by}.
	EventTaskListShared EventType = "tasklist.shared"
	// EventTaskListUnshared is published when a share is revoked.
	// Payload: {list_id, shared_with, removed_by}.
	EventTaskListUnshared EventType = "tasklist.unshared"
	// EventTaskMerged is published when a CRDT merge resolves for a task. The
	// payload is the winning TaskResponse so clients can reconcile.
	EventTaskMerged EventType = "task.merged"
	// EventPresenceOnline is published when a user connects to the sync hub.
	// Payload: {user_id, list_ids[]}.
	EventPresenceOnline EventType = "presence.online"
	// EventPresenceOffline is published when a user disconnects.
	// Payload: {user_id}.
	EventPresenceOffline EventType = "presence.offline"

	// Phase 9a: pomodoro lifecycle events (consumed by the insights module).
	// EventPomodoroSessionStarted: {session_id, user_id, focus_minutes, started_at}.
	EventPomodoroSessionStarted EventType = "pomodoro.session.started"
	// EventPomodoroSessionCompleted: {session_id, user_id, focus_minutes, elapsed_s, completed_at}.
	EventPomodoroSessionCompleted EventType = "pomodoro.session.completed"
	// EventPomodoroSessionCancelled: {session_id, user_id}.
	EventPomodoroSessionCancelled EventType = "pomodoro.session.cancelled"
)

// Event is the wire envelope pushed to subscribers and, ultimately, over the
// WebSocket to connected clients.
//
// Seq is assigned by the Bus implementation and is guaranteed to be strictly
// increasing within a single process, which lets clients resume after a
// disconnect without missing updates (see docs/adr/004-websocket-consistency.md).
//
// TraceID/SpanID carry the OpenTelemetry trace context of the producer (e.g.
// the HTTP request that created the task). They are not serialized to
// WebSocket clients; the Phase 3 RabbitMQ adapter will read them to propagate
// the trace through broker message headers.
type Event struct {
	// ID uniquely identifies the event instance. The in-memory bus leaves it
	// empty; the RabbitMQ adapter sets it to the AMQP message ID so consumers
	// can deduplicate at-least-once redeliveries.
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

// Handler receives a single event. Handlers MUST NOT block for long periods:
// the in-memory implementation invokes handlers synchronously in publish order,
// so a slow handler would stall publishers. Handlers must be safe for concurrent
// use (the same handler instance can be invoked from multiple publishers).
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
