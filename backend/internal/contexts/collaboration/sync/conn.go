package sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// staleEventType tells a reconnecting client that it fell too far behind the
// replay buffer and must refetch the full state via REST before resuming
// real-time updates.
const staleEventType eventbus.EventType = "stale"

// writeTimeout bounds each individual WebSocket write.
const writeTimeout = 10 * time.Second

// client wraps a single WebSocket connection. All writes are serialized through
// sendCh; dispatch never blocks the event bus.
type client struct {
	hub    *Hub
	conn   *websocket.Conn
	remote string

	// Phase 8: identity + membership. userID is the authenticated user from
	// the dev identity headers; role their application role. listIDs is the
	// set of task lists the user can access, resolved at connect time and
	// refreshed when a tasklist.shared/unshared event arrives. It is guarded
	// by mu (refreshed by the bus goroutine, read by the write pump).
	userID  string
	role    string
	mu      sync.RWMutex
	listIDs map[string]struct{}

	lastSeq int64
	sendCh  chan eventbus.Event

	closeOnce sync.Once
	closed    chan struct{}
}

func newClient(hub *Hub, conn *websocket.Conn, remote string, lastSeq int64, userID, role string, listIDs map[string]struct{}) *client {
	if listIDs == nil {
		listIDs = make(map[string]struct{})
	}
	return &client{
		hub:     hub,
		conn:    conn,
		remote:  remote,
		userID:  userID,
		role:    role,
		listIDs: listIDs,
		lastSeq: lastSeq,
		sendCh:  make(chan eventbus.Event, hub.cfg.ClientSendBuffer),
		closed:  make(chan struct{}),
	}
}

// hasAnyList reports whether the client's membership includes any of the given
// list IDs. Used to scope event dispatch to members of a list (Phase 8).
func (c *client) hasAnyList(listIDs []string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, id := range listIDs {
		if _, ok := c.listIDs[id]; ok {
			return true
		}
	}
	return false
}

// setListIDs atomically replaces the client's membership set.
func (c *client) setListIDs(listIDs map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listIDs = listIDs
}

// listIDSlice returns a copy of the client's membership as a slice.
func (c *client) listIDSlice() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.listIDs))
	for id := range c.listIDs {
		ids = append(ids, id)
	}
	return ids
}

// dispatch queues an event for delivery without blocking. A client that cannot
// keep up is disconnected so it can reconnect and replay from its last seq.
func (c *client) dispatch(event eventbus.Event) {
	select {
	case <-c.closed:
		return
	case c.sendCh <- event:
	default:
		log.Warn().Str("remote", c.remote).Msg("ws client too slow, disconnecting")
		c.close()
	}
}

// close idempotently tears down the connection.
func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.conn.Close(websocket.StatusNormalClosure, "bye")
	})
}

// run streams events to the connection. The write pump first sends the replay
// window (events published before this client registered, seq > lastSeq), then
// streams live events. Because register() adds the client to the registry at
// the same time it snapshots the replay, live events always have a higher seq
// than replay events, so ordering is preserved.
func (c *client) run(ctx context.Context, replay []eventbus.Event, stale bool) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); c.readPump(ctx) }()
	go func() { defer wg.Done(); c.writePump(ctx, replay, stale) }()
	wg.Wait()
}

// readPump consumes frames so control frames (ping/pong/close) are processed.
// Clients do not send data messages.
func (c *client) readPump(ctx context.Context) {
	for {
		if _, _, err := c.conn.Read(ctx); err != nil {
			c.close()
			return
		}
	}
}

// writePump sends the replay window, then drains sendCh (live events) to the
// connection.
func (c *client) writePump(ctx context.Context, replay []eventbus.Event, stale bool) {
	defer c.close()

	for _, e := range replay {
		targets, restricted := targetsFor(e)
		if restricted && !c.hasAnyList(targets) {
			continue
		}
		if !c.writeEvent(ctx, e) {
			return
		}
	}
	if stale {
		if !c.writeEvent(ctx, eventbus.Event{
			Type:      staleEventType,
			Timestamp: time.Now().UTC(),
		}) {
			return
		}
	}

	for {
		select {
		case <-c.closed:
			return
		case <-ctx.Done():
			return
		case event := <-c.sendCh:
			if !c.writeEvent(ctx, event) {
				return
			}
		}
	}
}

// writeEvent marshals and writes a single event frame. Returns false on error.
func (c *client) writeEvent(ctx context.Context, event eventbus.Event) bool {
	data, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal ws event")
		return true // skip malformed event but keep the connection
	}
	wctx, cancel := context.WithTimeout(ctx, writeTimeout)
	err = c.conn.Write(wctx, websocket.MessageText, data)
	cancel()
	if err != nil {
		log.Debug().Err(err).Str("remote", c.remote).Msg("ws write failed, closing")
		return false
	}
	return true
}
