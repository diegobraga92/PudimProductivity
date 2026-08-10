package httpclient

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedOpensAfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !cb.allow(time.Now()) {
			t.Fatalf("request %d should be allowed while closed", i+1)
		}
		cb.recordResult(false)
	}

	if cb.allow(time.Now()) {
		t.Fatal("circuit should be open after 3 consecutive failures")
	}
}

func TestCircuitBreaker_OpenRejectsThenHalfOpenProbes(t *testing.T) {
	cb := newCircuitBreaker(2, 100*time.Millisecond)
	cb.recordResult(false)
	cb.recordResult(false)

	if cb.allow(time.Now()) {
		t.Fatal("should reject while open")
	}

	// After openTimeout the circuit goes half-open and lets one probe through.
	if !cb.allow(time.Now().Add(200 * time.Millisecond)) {
		t.Fatal("probe should be allowed after openTimeout")
	}
	if cb.allow(time.Now().Add(200 * time.Millisecond)) {
		t.Fatal("only one probe should be in flight while half-open")
	}
}

func TestCircuitBreaker_HalfOpenSuccessRecloses(t *testing.T) {
	cb := newCircuitBreaker(1, 100*time.Millisecond)
	cb.recordResult(false) // open

	probeTime := time.Now().Add(200 * time.Millisecond)
	if !cb.allow(probeTime) {
		t.Fatal("probe should be allowed")
	}
	cb.recordResult(true)

	// Closed again: failures counter reset, requests flow.
	if !cb.allow(time.Now()) {
		t.Fatal("should be closed after successful probe")
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := newCircuitBreaker(1, 100*time.Millisecond)
	cb.recordResult(false) // open

	probeTime := time.Now().Add(200 * time.Millisecond)
	if !cb.allow(probeTime) {
		t.Fatal("probe should be allowed")
	}
	cb.recordResult(false)

	if cb.allow(time.Now().Add(50 * time.Millisecond)) {
		t.Fatal("should re-open immediately after failed probe")
	}
}
