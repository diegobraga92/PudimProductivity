package eventbus

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// CompositeBus fans Publish out to multiple underlying buses concurrently. It
// implements Bus, so producers (e.g. the task service) keep a single reference
// while the real-time sync hub and the async RabbitMQ pipeline each get their
// own copy of every event.
//
// Publish never blocks or fails because of a slow or unavailable child bus:
// each child is invoked in its own goroutine and errors are only logged. This
// is the core of the Phase 3 graceful-degradation story — a down RabbitMQ must
// not delay WebSocket fan-out.
//
// Subscribe on the composite is intentionally unsupported; consumers subscribe
// to the specific child bus they care about.
type CompositeBus struct {
	buses []Bus
}

// NewCompositeBus wraps one or more buses. A nil/empty set is allowed (all
// publishes become no-ops), which keeps degraded startups simple.
func NewCompositeBus(buses ...Bus) *CompositeBus {
	return &CompositeBus{buses: buses}
}

// Publish delivers the event to every child bus concurrently. Returns nil
// always — child failures are logged, never propagated to the caller.
func (c *CompositeBus) Publish(ctx context.Context, typ EventType, payload interface{}) error {
	var wg sync.WaitGroup
	for _, b := range c.buses {
		wg.Add(1)
		go func(child Bus) {
			defer wg.Done()
			if err := child.Publish(ctx, typ, payload); err != nil {
				log.Warn().Err(err).Str("event_type", string(typ)).
					Msg("composite bus: child publish failed (continuing)")
			}
		}(b)
	}
	wg.Wait()
	return nil
}

// Subscribe is not supported on the composite. Use the child bus you want to
// consume from directly.
func (c *CompositeBus) Subscribe(ctx context.Context, handler Handler) (func(), error) {
	return func() {}, nil
}

// Close closes every child bus.
func (c *CompositeBus) Close() error {
	for _, b := range c.buses {
		if err := b.Close(); err != nil {
			log.Warn().Err(err).Msg("composite bus: child close failed")
		}
	}
	return nil
}
