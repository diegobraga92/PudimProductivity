package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_GetReturnsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(Config{})
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestClient_NonRetryableStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(Config{Retries: 3})
	_, err := c.Get(context.Background(), srv.URL)
	var se *StatusError
	if !errors.As(err, &se) {
		t.Fatalf("want *StatusError, got %T: %v", err, err)
	}
	if se.Status != http.StatusNotFound {
		t.Fatalf("want 404, got %d", se.Status)
	}
}

func TestClient_RetriesThenFails(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{Retries: 2, Backoff: time.Millisecond})
	_, err := c.Get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if calls.Load() != 3 { // initial + 2 retries
		t.Fatalf("want 3 calls, got %d", calls.Load())
	}
}

func TestClient_RetrySucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "burst", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New(Config{Retries: 2, Backoff: time.Millisecond})
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get after retry: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestClient_CircuitOpensAndFailsFast(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	// MaxOpenFailures=3, no retries: 3 requests trip the breaker, then fast-fail.
	c := New(Config{MaxOpenFailures: 3, OpenTimeout: time.Hour})
	for i := 0; i < 3; i++ {
		if _, err := c.Get(context.Background(), srv.URL); err == nil {
			t.Fatalf("request %d should fail", i+1)
		}
	}
	before := calls.Load()
	if _, err := c.Get(context.Background(), srv.URL); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("want ErrCircuitOpen, got %v", err)
	}
	if calls.Load() != before {
		t.Fatal("open circuit must not reach the upstream server")
	}
}

func TestClient_ResponseSizeCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 1024)))
	}))
	defer srv.Close()

	c := New(Config{MaxResponseBytes: 512})
	if _, err := c.Get(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error when response exceeds MaxResponseBytes")
	}
}

func TestClient_ContextCancellationStopsRetryLoop(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := New(Config{Retries: 10, Backoff: time.Second})
	if _, err := c.Get(ctx, srv.URL); err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() > 3 {
		t.Fatalf("retry loop should stop quickly on cancellation, got %d calls", calls.Load())
	}
}
