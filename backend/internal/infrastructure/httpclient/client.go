// Package httpclient provides a small, testable HTTP client wrapper for
// external API integrations.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout         = 10 * time.Second
	defaultMaxResponse     = 4 << 20 // 4 MiB
	defaultBackoff         = 200 * time.Millisecond
	defaultMaxOpenFailures = 3
	defaultOpenTimeout     = 30 * time.Second
)

// Config tunes a Client. Zero values select the documented defaults.
type Config struct {
	// Timeout is the per-request timeout.
	Timeout time.Duration
	// MaxResponseBytes caps the response body read into memory.
	MaxResponseBytes int64
	// Retries is the number of additional attempts after a transient failure.
	Retries int
	// Backoff is the base delay between retries, doubled each attempt.
	Backoff time.Duration
	// Rate is the token-bucket refill rate in requests/second (0 = unlimited).
	Rate float64
	// Burst is the token-bucket burst size.
	Burst int
	// MaxOpenFailures opens the circuit after this many consecutive failures.
	MaxOpenFailures int
	// OpenTimeout is how long the circuit stays open.
	OpenTimeout time.Duration
}

// Client is a configured HTTP client. Safe for concurrent use.
type Client struct {
	http    *http.Client
	config  Config
	limiter *rateLimiter
	breaker *circuitBreaker
}

// ErrCircuitOpen is returned when the circuit breaker is open (fail fast).
var ErrCircuitOpen = errors.New("httpclient: circuit breaker open")

// StatusError carries the HTTP status of a non-2xx response.
type StatusError struct {
	Status int
	Body   []byte
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("httpclient: unexpected status %d", e.Status)
}

// New builds a Client applying defaults to zero-valued Config fields.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultMaxResponse
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = defaultBackoff
	}
	if cfg.MaxOpenFailures <= 0 {
		cfg.MaxOpenFailures = defaultMaxOpenFailures
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = defaultOpenTimeout
	}
	return &Client{
		http:    &http.Client{Timeout: cfg.Timeout},
		config:  cfg,
		limiter: newRateLimiter(cfg.Rate, cfg.Burst),
		breaker: newCircuitBreaker(cfg.MaxOpenFailures, cfg.OpenTimeout),
	}
}

// Get performs a GET request and returns the bounded response body.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	return c.Do(ctx, http.MethodGet, url, nil)
}

// Do performs a request through the rate limiter, circuit breaker and retry loop.
func (c *Client) Do(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	if err := c.limiter.wait(ctx); err != nil {
		return nil, err
	}
	if !c.breaker.allow(time.Now()) {
		return nil, ErrCircuitOpen
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.Retries; attempt++ {
		respBody, err := c.doOnce(ctx, method, url, body)
		if err == nil {
			c.breaker.recordResult(true)
			return respBody, nil
		}
		lastErr = err

		if !isRetryable(err) || attempt == c.config.Retries {
			break
		}
		backoff := c.config.Backoff * time.Duration(1<<uint(attempt))
		select {
		case <-ctx.Done():
			c.breaker.recordResult(false)
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	c.breaker.recordResult(false)
	return nil, lastErr
}

func (c *Client) doOnce(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytesReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Bound the read so a misbehaving third party cannot exhaust memory.
	limited := io.LimitReader(resp.Body, c.config.MaxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > c.config.MaxResponseBytes {
		return nil, errors.New("httpclient: response body exceeds MaxResponseBytes")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &StatusError{Status: resp.StatusCode, Body: data}
	}
	return data, nil
}

func bytesReader(b []byte) io.Reader {
	return readerFunc(func(p []byte) (int, error) {
		if len(b) == 0 {
			return 0, io.EOF
		}
		n := copy(p, b)
		b = b[n:]
		return n, nil
	})
}

type readerFunc func(p []byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

// isRetryable reports whether an error is safe to retry.
func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.Status == http.StatusTooManyRequests || se.Status >= 500
	}
	// Network-level failures (dial, read, TLS) are retryable.
	return true
}
