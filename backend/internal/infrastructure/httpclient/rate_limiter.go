package httpclient

import (
	"context"
	"sync"
	"time"
)

// rateLimiter is a thread-safe token bucket. A nil *rateLimiter (Rate == 0)
// means "no limit".
type rateLimiter struct {
	mu      sync.Mutex
	tokens  float64
	rate    float64 // tokens per second
	burst   float64
	lastRef time.Time
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	if rate <= 0 {
		return nil
	}
	if burst <= 0 {
		burst = int(rate)
	}
	return &rateLimiter{
		tokens:  float64(burst),
		rate:    rate,
		burst:   float64(burst),
		lastRef: time.Now(),
	}
}

// wait blocks until a token is available, or returns ctx.Err() when cancelled.
func (rl *rateLimiter) wait(ctx context.Context) error {
	if rl == nil {
		return nil
	}
	for {
		rl.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(rl.lastRef).Seconds()
		rl.tokens += elapsed * rl.rate
		if rl.tokens > rl.burst {
			rl.tokens = rl.burst
		}
		rl.lastRef = now

		if rl.tokens >= 1 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Sleep until the next token would be available.
		need := time.Duration(((1 - rl.tokens) / rl.rate) * float64(time.Second))
		rl.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(need):
		}
	}
}
