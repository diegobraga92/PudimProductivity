package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newTestRedisBus(t *testing.T, instanceID string) (*RedisBus, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	b, err := NewRedisBus(context.Background(), RedisConfig{
		URL:        "redis://" + s.Addr(),
		InstanceID: instanceID,
	})
	if err != nil {
		t.Fatalf("NewRedisBus: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b, s
}

func TestRedisBusCrossInstanceFanout(t *testing.T) {
	s := miniredis.RunT(t)
	producer, err := NewRedisBus(context.Background(), RedisConfig{
		URL:        "redis://" + s.Addr(),
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("NewRedisBus(producer): %v", err)
	}
	defer func() { _ = producer.Close() }()

	consumer, err := NewRedisBus(context.Background(), RedisConfig{
		URL:        "redis://" + s.Addr(),
		InstanceID: "instance-b",
	})
	if err != nil {
		t.Fatalf("NewRedisBus(consumer): %v", err)
	}
	defer func() { _ = consumer.Close() }()

	got := make(chan Event, 1)
	if _, err := consumer.Subscribe(context.Background(), func(_ context.Context, e Event) error {
		got <- e
		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Give the subscriber goroutine time to register with Redis before
	// publishing (pub/sub has no back-pressure for messages sent pre-subscribe).
	time.Sleep(150 * time.Millisecond)

	if err := producer.Publish(context.Background(), EventTaskUpdated, map[string]any{"id": "t1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case e := <-got:
		if e.Type != EventTaskUpdated {
			t.Fatalf("type = %q, want %q", e.Type, EventTaskUpdated)
		}
		payload, ok := e.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T, want map[string]any", e.Payload)
		}
		if payload["id"] != "t1" {
			t.Fatalf("payload id = %v, want t1", payload["id"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cross-instance event")
	}
}

func TestRedisBusFiltersSelfOrigin(t *testing.T) {
	b, _ := newTestRedisBus(t, "instance-a")

	calls := 0
	if _, err := b.Subscribe(context.Background(), func(_ context.Context, e Event) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	own, _ := json.Marshal(wireMessage{Origin: "instance-a", Event: Event{Type: EventTaskCreated}})
	b.handleMessage(string(own))
	if calls != 0 {
		t.Fatalf("self-origin message was dispatched (%d handler calls)", calls)
	}

	remote, _ := json.Marshal(wireMessage{Origin: "instance-b", Event: Event{Type: EventTaskDeleted}})
	b.handleMessage(string(remote))
	if calls != 1 {
		t.Fatalf("remote message was not dispatched (%d handler calls)", calls)
	}
}

func TestRedisBusDropsMalformedMessages(t *testing.T) {
	b, _ := newTestRedisBus(t, "instance-a")

	calls := 0
	if _, err := b.Subscribe(context.Background(), func(_ context.Context, e Event) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	b.handleMessage("this is not json")
	if calls != 0 {
		t.Fatalf("malformed message was dispatched (%d handler calls)", calls)
	}
}

func TestRedisBusClose(t *testing.T) {
	b, _ := newTestRedisBus(t, "instance-a")
	if _, err := b.Subscribe(context.Background(), func(_ context.Context, e Event) error { return nil }); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := b.Publish(context.Background(), EventTaskCreated, nil); err != ErrBusClosed {
		t.Fatalf("Publish after Close = %v, want ErrBusClosed", err)
	}
}
