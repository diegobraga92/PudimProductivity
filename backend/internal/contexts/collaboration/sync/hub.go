// Sync package is responsible for handling off-line syncing and sharing sync
package sync

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// TODO: Check about making default values more explicit instead of inside the constructor

// Config tunes the sync hub.
type Config struct {
	// ReplayBufferSize is the number of recent events retained for reconnecting clients.
	ReplayBufferSize int
	// ClientSendBuffer is the per-connection outbound channel capacity.
	ClientSendBuffer int
	// MaxClients caps concurrent WebSocket connections. 0 = unlimited.
	MaxClients int
}

// Hub fans bus events out to connected WebSocket clients and serves catch-up
// replays to reconnecting clients.
type Hub struct {
	bus      eventbus.Bus
	cfg      Config
	replay   *replayBuffer
	unsub    func()
	resolver MembershipResolver

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

// SetMembershipResolver configures membership resolution. Must be called before Start.
// When no resolver is set, broadcasts every event and presence endpoints report no users.
func (h *Hub) SetMembershipResolver(resolver MembershipResolver) {
	h.resolver = resolver
}

// Start subscribes the hub to the bus so it receives and replays events.
func (h *Hub) Start(ctx context.Context) error {
	unsub, err := h.bus.Subscribe(ctx, h.handleEvent)
	if err != nil {
		return err
	}
	h.unsub = unsub
	return nil
}

func (h *Hub) handleEvent(ctx context.Context, event eventbus.Event) error {
	h.mu.Lock()
	h.replay.push(event)
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	// When membership changes, refresh the related live connections.
	switch event.Type {
	case eventbus.EventTaskListShared, eventbus.EventTaskListUnshared:
		if m, ok := event.Payload.(map[string]any); ok {
			if uid, ok := m["shared_with"].(string); ok && uid != "" {
				h.refreshMembership(ctx, uid)
			}
		}
	}

	targets, restricted := affectedListIDs(event)
	for _, c := range clients {
		if restricted && !c.hasAnyList(targets) {
			continue
		}
		// TODO: Check if dispatch blocking can be a problem
		c.dispatch(event)
	}
	return nil
}

func (h *Hub) refreshMembership(ctx context.Context, userID string) {
	if h.resolver == nil {
		return
	}
	h.mu.Lock()
	var target []*client
	for c := range h.clients {
		if c.userID == userID {
			target = append(target, c)
		}
	}
	h.mu.Unlock()
	if len(target) == 0 {
		return
	}

	// TODO: Check if using target[0] here is reliable
	ids, err := h.resolver.ListIDsForUser(ctx, userID, target[0].role)
	if err != nil {
		// TODO: Just warning here might not be enough, check about forcing reconnect instead
		log.Warn().Err(err).Str("user_id", userID).Msg("failed to refresh membership")
		return
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	for _, c := range target {
		c.setListIDs(set)
	}
	log.Debug().Str("user_id", userID).Int("lists", len(set)).Msg("refreshed ws membership")
}

// OnlineUsersForList returns the distinct user IDs currently connected and able
// to access the given task list.
func (h *Hub) OnlineUsersForList(listID string) []string {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	seen := make(map[string]struct{})
	var users []string
	for _, c := range clients {
		if c.userID == "" || c.userID == "anonymous" {
			continue
		}
		if c.hasAnyList([]string{listID}) {
			if _, ok := seen[c.userID]; !ok {
				seen[c.userID] = struct{}{}
				users = append(users, c.userID)
			}
		}
	}
	return users
}

func (h *Hub) publishPresence(ctx context.Context, typ eventbus.EventType, payload any) {
	if err := h.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish presence event")
	}
}

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

// Close stops the hub: unsubscribes from the bus and shuts down all connected.
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
