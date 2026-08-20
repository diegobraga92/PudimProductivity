package scoringapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/httpclient"
)

func ip(v int) *int { return &v }

// omdbTestServer serves the search (s=) and detail (i=) endpoints OMDb exposes.
// detail returns the rating for every title; set wantNotFound to simulate OMDb
// reporting no matches.
func omdbTestServer(t *testing.T, wantNotFound bool) (*httptest.Server, *string) {
	t.Helper()
	var lastSearch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Query().Get("s") != "":
			lastSearch = r.URL.String()
			if wantNotFound {
				_ = json.NewEncoder(w).Encode(map[string]string{"Response": "False", "Error": "Movie not found!"})
				return
			}
			_ = json.NewEncoder(w).Encode(omdbSearchResponse{
				Search: []struct {
					Title  string `json:"Title"`
					Year   string `json:"Year"`
					ImdbID string `json:"imdbID"`
				}{
					{Title: "The Matrix", Year: "1999", ImdbID: "tt0133093"},
					{Title: "The Matrix Reloaded", Year: "2003", ImdbID: "tt0234215"},
				},
				Response: "True",
			})
		case r.URL.Query().Get("i") != "":
			_ = json.NewEncoder(w).Encode(omdbDetailResponse{ImdbRating: "8.7", Response: "True"})
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &lastSearch
}

func TestOMDB_MissingKey(t *testing.T) {
	if _, err := NewOMDB(context.Background(), ProviderConfig{}); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestOMDB_Search_HappyPath(t *testing.T) {
	srv, lastSearch := omdbTestServer(t, false)
	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}

	year := 1999
	cands, err := client.Search(context.Background(), library.ScoreQuery{
		Name: "Matrix", MediaType: library.MediaTypeMovie, ReleaseYear: &year,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(cands) != 2 {
		t.Fatalf("candidates = %d, want 2", len(cands))
	}
	if cands[0].Title != "The Matrix" || cands[0].Score != 8.7 || cands[0].Source != "imdb" ||
		cands[0].Year != 1999 || cands[0].ExternalID != "tt0133093" || cands[0].URL == "" {
		t.Fatalf("unexpected candidate: %+v", cands[0])
	}
	if !strings.Contains(*lastSearch, "type=movie") || !strings.Contains(*lastSearch, "y=1999") || !strings.Contains(*lastSearch, "apikey=k") {
		t.Fatalf("search request missing params: %s", *lastSearch)
	}
}

func TestOMDB_Search_NotFound_ReturnsEmpty(t *testing.T) {
	srv, _ := omdbTestServer(t, true)
	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "zzz", MediaType: library.MediaTypeMovie})
	if err != nil || len(cands) != 0 {
		t.Fatalf("Search(not found) = %v (err %v), want empty", cands, err)
	}
}

func TestOMDB_Search_RetriesServerError(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(omdbSearchResponse{Response: "False", Error: "Movie not found!"})
	}))
	t.Cleanup(srv.Close)

	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie}); err != nil {
		t.Fatalf("Search after transient 500: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2 (one retry)", attempts)
	}
}

func TestOMDB_Search_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie}); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestOMDB_Search_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "bad", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie}); err == nil || !strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("expected invalid API key error, got %v", err)
	}
}

func TestOMDB_Search_CircuitOpensAfterFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client, err := NewOMDB(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewOMDB: %v", err)
	}
	query := library.ScoreQuery{Name: "x", MediaType: library.MediaTypeMovie}

	// 3 consecutive failures trip the shared circuit breaker.
	for i := 0; i < 3; i++ {
		if _, err := client.Search(context.Background(), query); err == nil {
			t.Fatal("expected provider failures to surface as errors")
		}
	}
	// The next call must fail fast with ErrCircuitOpen.
	if _, err := client.Search(context.Background(), query); !errors.Is(err, httpclient.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen after 3 failures, got %v", err)
	}
}
