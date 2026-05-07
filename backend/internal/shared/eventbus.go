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
// It fans out events to all registered handlers synchronously.
// In Phase 3, this will be replaced with a RabbitMQ adapter.
type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// NewInMemoryEventBus creates a new InMemoryEventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish publishes a domain event to all subscribed handlers.
// Handlers are invoked synchronously in a goroutine to avoid blocking the publisher.
func (b *InMemoryEventBus) Publish(ctx context.Context, event DomainEvent) error {
	b.mu.RLock()
	handlers, ok := b.handlers[event.EventName()]
	b.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		log.Debug().Str("event", event.EventName()).Msg("no handlers registered for event")
		return nil
	}

	for _, handler := range handlers {
		h := handler // capture for goroutine
		go func() {
			if err := h(ctx, event); err != nil {
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

// Close shuts down the event bus.
func (b *InMemoryEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[string][]EventHandler)
	log.Info().Msg("in-memory event bus closed")
	return nil
}
