package sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/rs/zerolog/log"
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
	hub     *Hub
	conn    *websocket.Conn
	remote  string
	lastSeq int64
	sendCh  chan eventbus.Event

	closeOnce sync.Once
	closed    chan struct{}
}

func newClient(hub *Hub, conn *websocket.Conn, remote string, lastSeq int64) *client {
	return &client{
		hub:     hub,
		conn:    conn,
		remote:  remote,
		lastSeq: lastSeq,
		sendCh:  make(chan eventbus.Event, hub.cfg.ClientSendBuffer),
		closed:  make(chan struct{}),
	}
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
