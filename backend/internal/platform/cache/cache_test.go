package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	s := miniredis.RunT(t)
	c, err := New(context.Background(), "redis://"+s.Addr(), time.Minute)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestCacheGetSetRoundTrip(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	type item struct {
		ID string
		N  int
	}
	in := item{ID: "x", N: 42}
	if err := c.Set(ctx, "k", in); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var out item
	hit, err := c.Get(ctx, "k", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected a cache hit")
	}
	if out != in {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestCacheMiss(t *testing.T) {
	c := newTestCache(t)
	var out string
	hit, err := c.Get(context.Background(), "missing", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected a cache miss for an unknown key")
	}
}

func TestCacheDel(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "k", 1); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Del(ctx, "k", "missing-key"); err != nil {
		t.Fatalf("Del: %v", err)
	}

	var out int
	hit, err := c.Get(ctx, "k", &out)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected a miss after Del")
	}
}

func TestCacheVersionBumpInvalidates(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	v, err := c.Version(ctx, "tasks")
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != 0 {
		t.Fatalf("initial version = %d, want 0", v)
	}

	nv, err := c.Bump(ctx, "tasks")
	if err != nil {
		t.Fatalf("Bump: %v", err)
	}
	if nv != 1 {
		t.Fatalf("version after bump = %d, want 1", nv)
	}

	v2, err := c.Version(ctx, "tasks")
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v2 != 1 {
		t.Fatalf("version = %d, want 1", v2)
	}

	// Namespaces are independent.
	if _, err := c.Bump(ctx, "library"); err != nil {
		t.Fatalf("Bump(library): %v", err)
	}
	v3, _ := c.Version(ctx, "tasks")
	if v3 != 1 {
		t.Fatalf("tasks version = %d, want 1 after bumping another namespace", v3)
	}
}
