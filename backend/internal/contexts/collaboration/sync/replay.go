package sync

import (
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

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
	// TODO: Check if this approach is reliable in case server dies and loses seq count (maybe use epoch?)
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
