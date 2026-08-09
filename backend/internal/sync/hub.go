package sync

import (
	"context"
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/rs/zerolog/log"
)

// Config tunes the sync hub.
type Config struct {
	// ReplayBufferSize is the number of recent events retained in-process for
	// reconnecting clients. Default 1000.
	ReplayBufferSize int
	// ClientSendBuffer is the per-connection outbound channel capacity. A
	// client that cannot keep up is disconnected and reconnects to replay.
	// Default 64.
	ClientSendBuffer int
	// MaxClients caps concurrent WebSocket connections. 0 = unlimited.
	MaxClients int
}

// Hub fans bus events out to connected WebSocket clients and serves catch-up
// replays to reconnecting clients.
//
// Concurrency: a single mutex guards the client registry, the closed flag, and
// the replay buffer. This makes registration (replay snapshot + add client)
// atomic with respect to handleEvent (replay push + client snapshot), which
// guarantees each event is delivered to a connection exactly once: events
// published before registration arrive via replay, events published after
// arrive via the live dispatch.
type Hub struct {
	bus     eventbus.Bus
	cfg     Config
	replay  *replayBuffer
	unsub   func()

	mu      sync.Mutex
	clients map[*client]struct{}
	closed  bool
}

// NewHub creates a hub bound to the given event bus.
func NewHub(bus eventbus.Bus, cfg Config) *Hub {
	if cfg.ReplayBufferSize <= 0 {
		cfg.ReplayBufferSize = 1000
	}
	if cfg.ClientSendBuffer <= 0 {
		cfg.ClientSendBuffer = 64
	}
	return &Hub{
		bus:     bus,
		cfg:     cfg,
		replay:  newReplayBuffer(cfg.ReplayBufferSize),
		clients: make(map[*client]struct{}),
	}
}

// Start subscribes the hub to the bus so it receives and replays events. Call
// before serving WebSocket connections.
func (h *Hub) Start(ctx context.Context) error {
	unsub, err := h.bus.Subscribe(ctx, h.handleEvent)
	if err != nil {
		return err
	}
	h.unsub = unsub
	return nil
}

// handleEvent is the bus subscriber: it records the event for replay and pushes
// it to every connected client. Invoked synchronously by the bus in seq order.
func (h *Hub) handleEvent(ctx context.Context, event eventbus.Event) error {
	h.mu.Lock()
	h.replay.push(event)
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		c.dispatch(event)
	}
	return nil
}

// register atomically computes the client's replay window and adds it to the
// registry, so no event can be lost or duplicated at connect time.
//
// Returns:
//   - replay: buffered events to send before the live stream (Seq > c.lastSeq).
//   - stale: true if the client is too far behind and must full-refresh via REST.
//   - added: false if the hub is closed or at max capacity.
func (h *Hub) register(c *client) (replay []eventbus.Event, stale, added bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, false, false
	}
	if h.cfg.MaxClients > 0 && len(h.clients) >= h.cfg.MaxClients {
		return nil, false, false
	}
	replay, ok := h.replay.after(c.lastSeq)
	h.clients[c] = struct{}{}
	return replay, !ok, true
}

func (h *Hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

// Close stops the hub: unsubscribes from the bus and shuts down all connected
// clients.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	if h.unsub != nil {
		h.unsub()
	}
	for _, c := range clients {
		c.close()
	}
	log.Info().Int("clients_closed", len(clients)).Msg("sync hub closed")
}
