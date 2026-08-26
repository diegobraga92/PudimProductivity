package eventbus

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// InMemoryBus dispatches events to subscribed handlers within the same process.
type InMemoryBus struct {
	// publishMu serializes the publish+deliver critical section so subscribers
	// always observe events in monotonic seq order, even under concurrent
	// publishers. Handlers must be fast (buffered channel / buffer append).
	publishMu sync.Mutex
	mu        sync.RWMutex
	subs      []*subscription
	seq       atomic.Int64
	closed    bool
}

type subscription struct {
	handler Handler
	active  bool
}

// NewInMemoryBus returns a ready-to-use in-memory event bus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{}
}

// Publish assigns the next sequence number and delivers the event to all
// active subscribers in registration order, synchronously. Publishes are
// serialized so delivery order always matches seq order.
func (b *InMemoryBus) Publish(ctx context.Context, typ EventType, payload interface{}) error {
	b.publishMu.Lock()
	defer b.publishMu.Unlock()

	event := Event{
		Type:      typ,
		Seq:       b.seq.Add(1),
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	// Stamp the event with the producer's trace context so downstream
	// consumers can continue the trace.
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		event.TraceID = sc.TraceID().String()
		event.SpanID = sc.SpanID().String()
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	subs := make([]*subscription, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()

	for _, sub := range subs {
		if !sub.active {
			continue
		}
		if err := sub.handler(ctx, event); err != nil {
			// A handler failure must not prevent delivery to other subscribers.
			log.Error().Err(err).Str("event_type", string(event.Type)).
				Int64("seq", event.Seq).Msg("event handler failed")
		}
	}
	return nil
}

// Subscribe registers a handler. The returned function unsubscribes and is
// safe to call multiple times.
func (b *InMemoryBus) Subscribe(ctx context.Context, handler Handler) (func(), error) {
	if handler == nil {
		return func() {}, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return func() {}, ErrBusClosed
	}

	sub := &subscription{handler: handler, active: true}
	b.subs = append(b.subs, sub)
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		sub.active = false
	}, nil
}

// Close stops the bus and prevents further publishes/subscribes.
func (b *InMemoryBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	b.subs = nil
	return nil
}
