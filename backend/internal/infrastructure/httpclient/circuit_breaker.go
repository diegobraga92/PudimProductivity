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

// circuitBreaker is a minimal in-process circuit breaker.
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

// allow reports whether a request may proceed.
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
