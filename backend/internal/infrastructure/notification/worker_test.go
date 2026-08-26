package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

type recordingPusher struct {
	mu   sync.Mutex
	sent []string
}

func (r *recordingPusher) SendPush(_ context.Context, token, title, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, title+" | "+body)
	return nil
}

func (r *recordingPusher) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sent)
}

func TestRenderNotification(t *testing.T) {
	cases := []struct {
		typ      eventbus.EventType
		payload  map[string]interface{}
		wantOK   bool
		wantBody string
	}{
		{eventbus.EventTaskCreated, map[string]interface{}{"title": "Buy milk"}, true, "Task “Buy milk” was created."},
		{eventbus.EventTaskUpdated, map[string]interface{}{"title": "Buy milk"}, true, "Task “Buy milk” was updated."},
		{eventbus.EventTaskDeleted, map[string]interface{}{"id": "x"}, true, "A task was removed from your list."},
		{eventbus.EventTaskCompleted, map[string]interface{}{"title": "Run", "completed_date": "2026-08-09"}, true, "“Run” done for 2026-08-09."},
		{eventbus.EventTaskUncompleted, map[string]interface{}{"title": "Run"}, true, "The completion for “Run” was removed."},
		{"unknown.event", nil, false, ""},
	}

	for _, c := range cases {
		title, body, ok := renderNotification(eventbus.Event{Type: c.typ, Payload: c.payload})
		if ok != c.wantOK {
			t.Errorf("%s: ok=%v want %v", c.typ, ok, c.wantOK)
			continue
		}
		if ok && body != c.wantBody {
			t.Errorf("%s: body=%q want %q (title=%q)", c.typ, body, c.wantBody, title)
		}
	}
}

func TestWorker_SendsOncePerEventID(t *testing.T) {
	bus := eventbus.NewInMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	pusher := &recordingPusher{}
	worker := NewWorker(bus, []PushDeliverer{pusher},
		NewMemoryRepo(), Recipients{PushToken: "tok-1"})

	ctx := context.Background()

	// Two deliveries of the SAME logical message (same AMQP message ID, as
	// happens with at-least-once redelivery from RabbitMQ).
	evt := eventbus.Event{
		ID:        "msg-1",
		Type:      eventbus.EventTaskCreated,
		Timestamp: time.Now().UTC(),
		Payload:   map[string]interface{}{"title": "Task A"},
	}

	if err := worker.handleEvent(ctx, evt); err != nil {
		t.Fatalf("first handleEvent: %v", err)
	}
	if pusher.count() != 1 {
		t.Fatalf("expected 1 push, got %d", pusher.count())
	}

	if err := worker.handleEvent(ctx, evt); err != nil {
		t.Fatalf("second handleEvent: %v", err)
	}
	if pusher.count() != 1 {
		t.Fatalf("duplicate redelivery should be deduped, got %d pushes", pusher.count())
	}
}

func TestWorker_HandlerErrorPropagatesForRetry(t *testing.T) {
	bus := eventbus.NewInMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	failing := failingPusher{}
	worker := NewWorker(bus, []PushDeliverer{failing},
		NewMemoryRepo(), Recipients{PushToken: "tok-1"})

	err := worker.handleEvent(context.Background(), eventbus.Event{
		ID:      "msg-fail",
		Type:    eventbus.EventTaskCreated,
		Payload: map[string]interface{}{"title": "T"},
	})
	if err == nil {
		t.Fatal("expected error so the message is retried via the DLQ")
	}
}

type failingPusher struct{}

func (failingPusher) SendPush(context.Context, string, string, string) error {
	return context.DeadlineExceeded
}
