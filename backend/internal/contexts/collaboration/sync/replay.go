package sync

import (
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// replayBuffer retains the most recent events so reconnecting clients can catch
// up from their last-seen sequence number (see docs/adr/004-websocket-consistency.md).
// It is bounded: when full, the oldest event is evicted.
type replayBuffer struct {
	mu     sync.Mutex
	events []eventbus.Event
	size   int
}

func newReplayBuffer(size int) *replayBuffer {
	if size < 1 {
		size = 1
	}
	return &replayBuffer{size: size}
}

// push appends an event, evicting the oldest when at capacity.
func (r *replayBuffer) push(event eventbus.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.events) == r.size {
		copy(r.events, r.events[1:])
		r.events = r.events[:r.size-1]
	}
	r.events = append(r.events, event)
}

// after returns every buffered event with Seq > seq, in order, plus whether a
// complete catch-up is possible.
//
//   - ok == true with a non-empty slice → client should apply these events.
//   - ok == true with an empty slice → client is fully up to date.
//   - ok == false → the requested seq is older than the oldest buffered event;
//     the caller must tell the client to do a full REST refresh.
func (r *replayBuffer) after(seq int64) (events []eventbus.Event, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.events) == 0 {
		return nil, true
	}
	oldest := r.events[0].Seq
	newest := r.events[len(r.events)-1].Seq

	if seq >= newest {
		return nil, true
	}
	// Events are contiguous (seq increments by 1 per publish); if the client's
	// last seq is more than one behind the oldest buffered event, at least one
	// event has been evicted and we cannot guarantee a complete replay.
	if seq < oldest-1 {
		return nil, false
	}
	for _, e := range r.events {
		if e.Seq > seq {
			events = append(events, e)
		}
	}
	return events, true
}
