package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/config"
)

func TestParseAllowedOrigins(t *testing.T) {
	got := config.ParseAllowedOrigins(" app://bundle , https://example.com ,")
	if !got["app://bundle"] {
		t.Error("expected app://bundle to be allowed")
	}
	if !got["https://example.com"] {
		t.Error("expected https://example.com to be allowed")
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 origins, got %d", len(got))
	}
	if config.ParseAllowedOrigins("") != nil && len(config.ParseAllowedOrigins("")) != 0 {
		t.Error("expected empty allow-list for empty input")
	}
}

func TestCorsMiddleware(t *testing.T) {
	origins := map[string]bool{"app://bundle": true}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed origin gets headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Origin", "app://bundle")
		rec := httptest.NewRecorder()
		CorsMiddleware(origins)(next).ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "app://bundle" {
			t.Errorf("expected Access-Control-Allow-Origin=app://bundle, got %q", got)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected handler to run for non-preflight, got %d", rec.Code)
		}
	})

	t.Run("preflight is answered with 204 and short-circuits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
		req.Header.Set("Origin", "app://bundle")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		CorsMiddleware(origins)(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 for preflight, got %d", rec.Code)
		}
	})

	t.Run("disallowed origin passes through without headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		CorsMiddleware(origins)(next).ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS headers for a disallowed origin")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected handler to still run, got %d", rec.Code)
		}
	})

	t.Run("no origins configured is a no-op", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Origin", "app://bundle")
		rec := httptest.NewRecorder()
		CorsMiddleware(nil)(next).ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS headers when no origins are configured")
		}
	})

	t.Run("missing Origin is a no-op", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()
		CorsMiddleware(origins)(next).ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no CORS headers when Origin is absent")
		}
	})
}
