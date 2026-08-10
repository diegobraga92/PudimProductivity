package googlebooks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/httpclient"
)

const sampleVolume = `{
	"totalItems": 1,
	"items": [{
		"volumeInfo": {
			"title": "Permanent Record",
			"authors": ["Edward Snowden"],
			"publisher": "Macmillan",
			"publishedDate": "2019-09-17",
			"description": "A memoir.",
			"pageCount": 352,
			"imageLinks": {"thumbnail": "http://books.google.com/thumb.jpg"}
		}
	}]
}`

func TestLookupByISBN_HappyPath(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(sampleVolume))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL})
	info, err := c.LookupByISBN(context.Background(), "9781250237231")
	if err != nil {
		t.Fatalf("LookupByISBN: %v", err)
	}
	if info.Title != "Permanent Record" || len(info.Authors) != 1 || info.PageCount != 352 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if !strings.Contains(gotQuery, "q=isbn%3A9781250237231") {
		t.Fatalf("query does not contain isbn lookup: %q", gotQuery)
	}
}

func TestLookupByISBN_NotFoundedReturnsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"totalItems": 0, "items": []}`))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL})
	_, err := c.LookupByISBN(context.Background(), "9780000000001")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestLookupByISBN_RetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(sampleVolume))
	}))
	defer srv.Close()

	c := NewClient(Config{
		BaseURL: srv.URL,
		HTTP:    httpclient.Config{Retries: 3, Backoff: time.Millisecond, Timeout: 5 * time.Second},
	})
	if _, err := c.LookupByISBN(context.Background(), "9781250237231"); err != nil {
		t.Fatalf("LookupByISBN after retries: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("want 3 calls, got %d", calls.Load())
	}
}

func TestLookupByISBN_CircuitBreakerFailsFast(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	// No retries; 3 consecutive failures trip the breaker.
	c := NewClient(Config{
		BaseURL: srv.URL,
		HTTP:    httpclient.Config{MaxOpenFailures: 3, OpenTimeout: time.Minute},
	})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := c.LookupByISBN(ctx, "9781250237231"); err == nil {
			t.Fatalf("request %d should fail", i+1)
		}
	}
	before := calls.Load()
	if _, err := c.LookupByISBN(ctx, "9781250237231"); !errors.Is(err, httpclient.ErrCircuitOpen) {
		t.Fatalf("want ErrCircuitOpen, got %v", err)
	}
	if calls.Load() != before {
		t.Fatal("open circuit must not reach the upstream server")
	}
}

func TestLookupByISBN_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL})
	if _, err := c.LookupByISBN(context.Background(), "9781250237231"); err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestLookupByISBN_SendsAPIKeyWhenConfigured(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(sampleVolume))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL, APIKey: "secret-key"})
	if _, err := c.LookupByISBN(context.Background(), "9781250237231"); err != nil {
		t.Fatalf("LookupByISBN: %v", err)
	}
	if !strings.Contains(gotQuery, "key=secret-key") {
		t.Fatalf("API key not sent: %q", gotQuery)
	}
}
