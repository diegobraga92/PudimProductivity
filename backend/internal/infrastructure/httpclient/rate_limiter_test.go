package httpclient

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_NoLimitWhenRateZero(t *testing.T) {
	rl := newRateLimiter(0, 0)
	if rl != nil {
		t.Fatal("rate 0 should produce a nil limiter (unlimited)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := (*rateLimiter)(nil).wait(ctx); err != nil {
		t.Fatalf("nil limiter should never block: %v", err)
	}
}

func TestRateLimiter_BurstAllowsImmediateTokens(t *testing.T) {
	rl := newRateLimiter(1, 5) // 1 token/sec, burst 5
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := rl.wait(ctx); err != nil {
			t.Fatalf("wait %d should not block within burst: %v", i, err)
		}
	}
	// Burst exhausted: the 6th call must block or time out.
	timeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := rl.wait(timeout); err == nil {
		t.Fatal("6th token should not be immediately available")
	}
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	rl := newRateLimiter(10, 1) // 10 tokens/sec, burst 1
	ctx := context.Background()
	if err := rl.wait(ctx); err != nil {
		t.Fatalf("first token should be available: %v", err)
	}
	// One token refills every 100ms.
	time.Sleep(150 * time.Millisecond)
	if err := rl.wait(ctx); err != nil {
		t.Fatalf("token should have refilled: %v", err)
	}
}

func TestRateLimiter_RespectsContextCancellation(t *testing.T) {
	rl := newRateLimiter(0.001, 1) // ~1 token every 1000s
	ctx := context.Background()
	// Consume the burst token first so the next wait actually blocks.
	if err := rl.wait(ctx); err != nil {
		t.Fatalf("first token should be available from burst: %v", err)
	}
	cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := rl.wait(cancelCtx); err == nil {
		t.Fatal("expected context cancellation while waiting for a token")
	}
}
