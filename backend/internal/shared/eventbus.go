package shared

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DomainEvent is the interface that all domain events must implement.
type DomainEvent interface {
	// EventName returns a unique name for the event type (e.g., "task.created").
	EventName() string
	// AggregateID returns the ID of the aggregate that emitted the event.
	AggregateID() string
	// OccurredAt returns the timestamp when the event occurred.
	OccurredAt() time.Time
}

// EventHandler is a function that handles a domain event.
type EventHandler func(ctx context.Context, event DomainEvent) error

// EventBus defines the interface for publishing and subscribing to domain events.
type EventBus interface {
	// Publish publishes a domain event to all subscribed handlers.
	Publish(ctx context.Context, event DomainEvent) error
	// Subscribe registers a handler for a specific event name.
	Subscribe(eventName string, handler EventHandler)
	// Close shuts down the event bus and waits for in-flight handlers to complete.
	Close() error
}

// InMemoryEventBus is an in-memory implementation of EventBus.
// It fans out events to all registered handlers asynchronously.
// In Phase 3, this will be replaced with a RabbitMQ adapter.
type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
	wg       sync.WaitGroup
	closed   bool
}

// NewInMemoryEventBus creates a new InMemoryEventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publishes a domain event to all subscribed handlers.
// Each handler runs in its own goroutine. The context passed to handlers is
// detached from the caller's context (cancellation and deadlines are stripped
// but values such as request IDs are preserved) so that a handler can outlive
// the HTTP request that triggered the event.
func (b *InMemoryEventBus) Publish(ctx context.Context, event DomainEvent) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		log.Warn().Str("event", event.EventName()).Msg("event bus is closed, dropping event")
		return nil
	}
	handlers, ok := b.handlers[event.EventName()]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		log.Debug().Str("event", event.EventName()).Msg("no handlers registered for event")
		return nil
	}

	// Detach from the caller's context so handler goroutines are not cancelled
	// when the HTTP request completes, while still preserving context values.
	detached := context.WithoutCancel(ctx)

	for _, handler := range handlers {
		h := handler
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			if err := h(detached, event); err != nil {
				log.Error().Err(err).Str("event", event.EventName()).Msg("event handler failed")
			}
		}()
	}

	return nil
}

// Subscribe registers a handler for a specific event name.
func (b *InMemoryEventBus) Subscribe(eventName string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
	log.Debug().Str("event", eventName).Msg("handler registered for event")
}

// Close stops accepting new events, waits for all in-flight handlers to finish,
// then releases resources.
func (b *InMemoryEventBus) Close() error {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()

	b.wg.Wait()

	b.mu.Lock()
	b.handlers = make(map[string][]EventHandler)
	b.mu.Unlock()

	log.Info().Msg("in-memory event bus closed")
	return nil
}
