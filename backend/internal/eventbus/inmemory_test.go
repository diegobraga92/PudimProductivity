package eventbus

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestInMemoryBus_PublishesToSubscriberInOrder(t *testing.T) {
	b := NewInMemoryBus()
	defer b.Close()

	ctx := context.Background()
	var received []Event
	unsub, err := b.Subscribe(ctx, func(_ context.Context, e Event) error {
		received = append(received, e)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	for i := 0; i < 5; i++ {
		if err := b.Publish(ctx, EventTaskCreated, map[string]any{"id": i}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	if len(received) != 5 {
		t.Fatalf("expected 5 events, got %d", len(received))
	}
	for i, e := range received {
		if e.Seq != int64(i+1) {
			t.Errorf("event %d has seq %d, want %d", i, e.Seq, i+1)
		}
		if e.Type != EventTaskCreated {
			t.Errorf("event %d type = %q", i, e.Type)
		}
	}
}

func TestInMemoryBus_MultiSubscriberFanout(t *testing.T) {
	b := NewInMemoryBus()
	defer b.Close()

	ctx := context.Background()
	var mu1, mu2 []Event
	unsub1, _ := b.Subscribe(ctx, func(_ context.Context, e Event) error { mu1 = append(mu1, e); return nil })
	unsub2, _ := b.Subscribe(ctx, func(_ context.Context, e Event) error { mu2 = append(mu2, e); return nil })
	defer unsub1()
	defer unsub2()

	if err := b.Publish(ctx, EventTaskUpdated, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(mu1) != 1 || len(mu2) != 1 {
		t.Fatalf("expected all subscribers to receive event, got %d and %d", len(mu1), len(mu2))
	}
}

func TestInMemoryBus_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewInMemoryBus()
	defer b.Close()

	ctx := context.Background()
	var received []Event
	unsub, _ := b.Subscribe(ctx, func(_ context.Context, e Event) error {
		received = append(received, e)
		return nil
	})

	_ = b.Publish(ctx, EventTaskCreated, nil)
	unsub()
	_ = b.Publish(ctx, EventTaskCreated, nil)

	if len(received) != 1 {
		t.Fatalf("expected 1 event after unsubscribe, got %d", len(received))
	}
}

func TestInMemoryBus_HandlerErrorDoesNotBlockOthers(t *testing.T) {
	b := NewInMemoryBus()
	defer b.Close()

	ctx := context.Background()
	errHandler := func(_ context.Context, e Event) error { return errors.New("boom") }
	_, _ = b.Subscribe(ctx, errHandler)

	var received []Event
	_, _ = b.Subscribe(ctx, func(_ context.Context, e Event) error { received = append(received, e); return nil })

	if err := b.Publish(ctx, EventTaskDeleted, nil); err != nil {
		t.Fatalf("publish should not surface handler error, got %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("healthy subscriber should still receive event, got %d", len(received))
	}
}

func TestInMemoryBus_PublishAfterCloseFails(t *testing.T) {
	b := NewInMemoryBus()
	if err := b.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := b.Publish(context.Background(), EventTaskCreated, nil); err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
	if _, err := b.Subscribe(context.Background(), func(context.Context, Event) error { return nil }); err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed on subscribe, got %v", err)
	}
}

func TestInMemoryBus_SeqMonotonicAcrossPublishers(t *testing.T) {
	b := NewInMemoryBus()
	defer b.Close()

	ctx := context.Background()
	var mu sync.Mutex
	var maxSeq int64
	_, _ = b.Subscribe(ctx, func(_ context.Context, e Event) error {
		mu.Lock()
		defer mu.Unlock()
		if e.Seq < maxSeq {
			t.Errorf("sequence regression: %d after %d", e.Seq, maxSeq)
		}
		maxSeq = e.Seq
		return nil
	})

	const goroutines = 8
	const perGoroutine = 25
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				_ = b.Publish(ctx, EventTaskCreated, nil)
			}
		}()
	}
	wg.Wait()
}
