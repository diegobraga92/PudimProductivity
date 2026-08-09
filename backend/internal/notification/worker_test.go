package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

type recordingEmailer struct {
	mu   sync.Mutex
	sent []string
}

func (r *recordingEmailer) SendEmail(_ context.Context, to, subject, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, subject+" | "+body)
	return nil
}

func (r *recordingEmailer) count() int {
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

	emailer := &recordingEmailer{}
	worker := NewWorker(bus, []EmailDeliverer{emailer}, []PushDeliverer{NoopSender{}},
		NewMemoryRepo(), Recipients{Email: "a@b.c"})

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
	if emailer.count() != 1 {
		t.Fatalf("expected 1 email, got %d", emailer.count())
	}

	if err := worker.handleEvent(ctx, evt); err != nil {
		t.Fatalf("second handleEvent: %v", err)
	}
	if emailer.count() != 1 {
		t.Fatalf("duplicate redelivery should be deduped, got %d emails", emailer.count())
	}
}

func TestWorker_HandlerErrorPropagatesForRetry(t *testing.T) {
	bus := eventbus.NewInMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })

	failing := failingEmailer{}
	worker := NewWorker(bus, []EmailDeliverer{failing}, nil,
		NewMemoryRepo(), Recipients{Email: "a@b.c"})

	err := worker.handleEvent(context.Background(), eventbus.Event{
		ID:      "msg-fail",
		Type:    eventbus.EventTaskCreated,
		Payload: map[string]interface{}{"title": "T"},
	})
	if err == nil {
		t.Fatal("expected error so the message is retried via the DLQ")
	}
}

type failingEmailer struct{}

func (failingEmailer) SendEmail(context.Context, string, string, string) error {
	return context.DeadlineExceeded
}
