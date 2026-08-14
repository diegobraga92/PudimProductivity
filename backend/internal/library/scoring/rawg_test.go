package scoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/library"
)

func TestRAWG_MissingKey(t *testing.T) {
	if _, err := NewRAWG(context.Background(), ProviderConfig{}); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestRAWG_Search_HappyPath(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rawgResponse{
			Results: []struct {
				ID         int    `json:"id"`
				Name       string `json:"name"`
				Released   string `json:"released"`
				Metacritic *int   `json:"metacritic"`
				Slug       string `json:"slug"`
			}{
				{ID: 1, Name: "Elden Ring", Released: "2022-02-25", Metacritic: ip(96), Slug: "elden-ring"},
				{ID: 2, Name: "Sekiro", Released: "2019-03-22", Metacritic: nil, Slug: "sekiro-shadows-die-twice"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client, err := NewRAWG(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewRAWG: %v", err)
	}

	year := 2022
	cands, err := client.Search(context.Background(), library.ScoreQuery{
		Name: "Elden Ring", MediaType: library.MediaTypeGame, ReleaseYear: &year,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(cands) != 1 { // Sekiro has no Metacritic rating → filtered
		t.Fatalf("candidates = %d, want 1 (unrated games filtered)", len(cands))
	}
	if cands[0].Title != "Elden Ring" || cands[0].Score != 96 || cands[0].Source != "metacritic" ||
		cands[0].Year != 2022 || cands[0].ExternalID != "1" || cands[0].URL != "https://rawg.io/games/elden-ring" {
		t.Fatalf("unexpected candidate: %+v", cands[0])
	}
	if !strings.Contains(lastQuery, "key=k") || !strings.Contains(lastQuery, "search=Elden+Ring") ||
		!strings.Contains(lastQuery, "dates=2022-01-01") {
		t.Fatalf("request missing params: %s", lastQuery)
	}
}

func TestRAWG_Search_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewRAWG(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewRAWG: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "zzz", MediaType: library.MediaTypeGame})
	if err != nil || len(cands) != 0 {
		t.Fatalf("Search(empty) = %v (err %v), want empty", cands, err)
	}
}

func TestRAWG_Search_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	client, err := NewRAWG(context.Background(), ProviderConfig{APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("NewRAWG: %v", err)
	}
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame}); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
