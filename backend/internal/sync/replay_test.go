package sync

import (
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

func evt(seq int64, typ eventbus.EventType) eventbus.Event {
	return eventbus.Event{Type: typ, Seq: seq, Timestamp: time.Now().UTC()}
}

func TestReplayBuffer_Empty(t *testing.T) {
	buf := newReplayBuffer(100)
	events, ok := buf.after(0)
	if !ok || len(events) != 0 {
		t.Fatalf("empty buffer: ok=%v events=%d, want ok=true events=0", ok, len(events))
	}
}

func TestReplayBuffer_ReplayAllFromZero(t *testing.T) {
	buf := newReplayBuffer(100)
	for i := int64(1); i <= 5; i++ {
		buf.push(evt(i, eventbus.EventTaskCreated))
	}
	events, ok := buf.after(0)
	if !ok || len(events) != 5 {
		t.Fatalf("want 5 events, got %d (ok=%v)", len(events), ok)
	}
	for i, e := range events {
		if e.Seq != int64(i+1) {
			t.Errorf("replay order broken: events[%d].Seq=%d", i, e.Seq)
		}
	}
}

func TestReplayBuffer_ReplayAfterSeq(t *testing.T) {
	buf := newReplayBuffer(100)
	for i := int64(1); i <= 10; i++ {
		buf.push(evt(i, eventbus.EventTaskUpdated))
	}
	events, ok := buf.after(7)
	if !ok || len(events) != 3 { // seq 8, 9, 10
		t.Fatalf("want 3 events after seq 7, got %d (ok=%v)", len(events), ok)
	}
	if events[0].Seq != 8 || events[2].Seq != 10 {
		t.Errorf("unexpected replay window: %d..%d", events[0].Seq, events[2].Seq)
	}
}

func TestReplayBuffer_UpToDate(t *testing.T) {
	buf := newReplayBuffer(100)
	for i := int64(1); i <= 5; i++ {
		buf.push(evt(i, eventbus.EventTaskDeleted))
	}
	events, ok := buf.after(5)
	if !ok || len(events) != 0 {
		t.Fatalf("want no events when up to date, got %d (ok=%v)", len(events), ok)
	}
	events, ok = buf.after(99)
	if !ok || len(events) != 0 {
		t.Fatalf("want no events when client ahead, got %d (ok=%v)", len(events), ok)
	}
}

func TestReplayBuffer_StaleWhenGap(t *testing.T) {
	buf := newReplayBuffer(5)
	for i := int64(1); i <= 5; i++ {
		buf.push(evt(i, eventbus.EventTaskCreated))
	}
	// Oldest retained is seq 1. A client last saw seq 0 → seq 0 == oldest-1,
	// so replay of all 5 is possible.
	if events, ok := buf.after(0); !ok || len(events) != 5 {
		t.Fatalf("seq 0 should replay all 5, got %d (ok=%v)", len(events), ok)
	}
	// But after overflow the oldest is evicted.
	buf.push(evt(6, eventbus.EventTaskCreated))
	// Now oldest = 2, so a client at seq 0 is more than one behind → stale.
	if _, ok := buf.after(0); ok {
		t.Fatal("seq 0 after overflow should be stale")
	}
	// A client at seq 1 (oldest-1) can still replay 2..6.
	if events, ok := buf.after(1); !ok || len(events) != 5 {
		t.Fatalf("seq 1 should replay 2..6, got %d (ok=%v)", len(events), ok)
	}
}

func TestReplayBuffer_OverflowEvictsOldest(t *testing.T) {
	buf := newReplayBuffer(3)
	for i := int64(1); i <= 5; i++ {
		buf.push(evt(i, eventbus.EventTaskCreated))
	}
	events, ok := buf.after(2)
	if !ok || len(events) != 3 {
		t.Fatalf("buffer should keep 3,4,5, got %d (ok=%v)", len(events), ok)
	}
	if events[0].Seq != 3 || events[2].Seq != 5 {
		t.Errorf("unexpected contents: %d..%d", events[0].Seq, events[2].Seq)
	}
}
