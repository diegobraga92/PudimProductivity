package httpclient

import (
	"sync"
	"time"
)

type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

// circuitBreaker is a minimal in-process circuit breaker:
//
//	closed:    requests flow; consecutive failures are counted.
//	open:      requests fail fast (ErrCircuitOpen) until openTimeout elapses.
//	half-open: one probe request is allowed; success re-closes, failure re-opens.
//
// Thread-safe and deliberately small — sized for a handful of external-API
// adapters inside the single backend process.
type circuitBreaker struct {
	mu          sync.Mutex
	state       breakerState
	failures    int
	maxFailures int
	openTimeout time.Duration
	openedAt    time.Time
}

func newCircuitBreaker(maxFailures int, openTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{maxFailures: maxFailures, openTimeout: openTimeout}
}

// allow reports whether a request may proceed. When the circuit transitions
// open → half-open, exactly one probe is let through.
func (cb *circuitBreaker) allow(now time.Time) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateOpen:
		if now.Sub(cb.openedAt) >= cb.openTimeout {
			cb.state = stateHalfOpen
			return true // the single probe request
		}
		return false
	case stateHalfOpen:
		return false // a probe is already in flight
	default:
		return true
	}
}

// recordResult records the outcome of a completed request.
func (cb *circuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateHalfOpen:
		if success {
			cb.state = stateClosed
			cb.failures = 0
		} else {
			cb.state = stateOpen
			cb.openedAt = time.Now()
		}
	case stateClosed:
		if success {
			cb.failures = 0
		} else {
			cb.failures++
			if cb.failures >= cb.maxFailures {
				cb.state = stateOpen
				cb.openedAt = time.Now()
			}
		}
	}
}
