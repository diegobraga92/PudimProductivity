package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

// startHub wires an InMemoryBus + Hub and exposes ServeHTTP via httptest.
func startHub(t *testing.T, cfg Config) (*eventbus.InMemoryBus, *Hub, *httptest.Server) {
	t.Helper()
	bus := eventbus.NewInMemoryBus()
	hub := NewHub(bus, cfg)
	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("hub start: %v", err)
	}
	t.Cleanup(func() {
		hub.Close()
		bus.Close()
	})

	srv := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	t.Cleanup(srv.Close)
	return bus, hub, srv
}

func wsURL(server *httptest.Server, lastSeq int64) string {
	// httptest uses http://; WebSocket needs ws://. last_seq query is optional.
	u := "ws" + strings.TrimPrefix(server.URL, "http")
	if lastSeq > 0 {
		u += "/?last_seq=" + strconv.FormatInt(lastSeq, 10)
	}
	return u
}

func TestHub_FanoutToConnectedClient(t *testing.T) {
	bus, _, srv := startHub(t, Config{ReplayBufferSize: 100})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv, 0), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test done")

	// Publish three events; the connected client should receive all of them in order.
	if err := bus.Publish(ctx, eventbus.EventTaskCreated, map[string]any{"id": "t1"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := bus.Publish(ctx, eventbus.EventTaskUpdated, map[string]any{"id": "t1"}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := bus.Publish(ctx, eventbus.EventTaskDeleted, map[string]any{"id": "t1"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	for i, wantType := range []eventbus.EventType{
		eventbus.EventTaskCreated, eventbus.EventTaskUpdated, eventbus.EventTaskDeleted,
	} {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read event %d: %v", i, err)
		}
		var envelope struct {
			Type  eventbus.EventType `json:"type"`
			Seq   int64              `json:"seq"`
			Bytes json.RawMessage    `json:"payload"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatalf("unmarshal event %d: %v (raw=%s)", i, err, data)
		}
		if envelope.Type != wantType {
			t.Errorf("event %d type = %q, want %q", i, envelope.Type, wantType)
		}
		if envelope.Seq != int64(i+1) {
			t.Errorf("event %d seq = %d, want %d", i, envelope.Seq, i+1)
		}
		if len(envelope.Bytes) == 0 {
			t.Errorf("event %d missing payload", i)
		}
	}
}

func TestHub_ReplayOnReconnect(t *testing.T) {
	bus, _, srv := startHub(t, Config{ReplayBufferSize: 100})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Produce events 1..5 before the client connects.
	for i := 0; i < 5; i++ {
		if err := bus.Publish(ctx, eventbus.EventTaskCreated, map[string]any{"id": "t"}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	// A fresh client (last_seq=0) must receive all 5 buffered events.
	conn, _, err := websocket.Dial(ctx, wsURL(srv, 0), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	for i := 1; i <= 5; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read replay event %d: %v", i, err)
		}
		var e eventbus.Event
		if err := json.Unmarshal(data, &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if e.Seq != int64(i) {
			t.Errorf("replay event seq = %d, want %d", e.Seq, i)
		}
	}

	// Close and reconnect with last_seq=5 → only newer events arrive.
	_ = conn.Close(websocket.StatusNormalClosure, "reconnect")
	if err := bus.Publish(ctx, eventbus.EventTaskUpdated, map[string]any{"id": "t"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	conn2, _, err := websocket.Dial(ctx, wsURL(srv, 5), nil)
	if err != nil {
		t.Fatalf("re-dial: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "done")

	_, data, err := conn2.Read(ctx)
	if err != nil {
		t.Fatalf("read after reconnect: %v", err)
	}
	var e eventbus.Event
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Seq != 6 {
		t.Errorf("after last_seq=5, first event seq = %d, want 6", e.Seq)
	}
}

func TestHub_StaleSignalWhenTooFarBehind(t *testing.T) {
	bus, _, srv := startHub(t, Config{ReplayBufferSize: 3})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 5; i++ {
		if err := bus.Publish(ctx, eventbus.EventTaskCreated, map[string]any{"id": "t"}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}
	// Oldest retained seq is now 3 (buffer size 3). A client at seq 1 is stale.
	conn, _, err := websocket.Dial(ctx, wsURL(srv, 1), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var e eventbus.Event
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != staleEventType {
		t.Fatalf("first message type = %q, want %q", e.Type, staleEventType)
	}
}
