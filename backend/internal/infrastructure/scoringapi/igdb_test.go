package scoringapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
)

func fp(v float64) *float64 { return &v }
func ip64(v int64) *int64   { return &v }

// igdbTestServer serves the Twitch OAuth token endpoint and the IGDB /games
// endpoint. It returns the number of token requests made.
func igdbTestServer(t *testing.T, games func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var tokenCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			tokenCalls.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("token: method = %s, want POST", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "client_id=cid") ||
				!strings.Contains(string(body), "client_secret=secret") ||
				!strings.Contains(string(body), "grant_type=client_credentials") {
				t.Errorf("token body missing credentials: %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(igdbTokenResponse{AccessToken: "tok123", ExpiresIn: 3600})
		case "/v4/games":
			games(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &tokenCalls
}

func igdbProvider(srv *httptest.Server) ProviderConfig {
	return ProviderConfig{
		APIKey:  "cid",
		BaseURL: srv.URL + "/v4",
		Settings: map[string]string{
			"client_secret": "secret",
			"token_url":     srv.URL + "/oauth2/token",
		},
	}
}

func TestIGDB_MissingClientID(t *testing.T) {
	if _, err := NewIGDB(context.Background(), ProviderConfig{Settings: map[string]string{"client_secret": "s"}}); err == nil {
		t.Fatal("expected error for missing client ID")
	}
}

func TestIGDB_MissingClientSecret(t *testing.T) {
	if _, err := NewIGDB(context.Background(), ProviderConfig{APIKey: "cid"}); err == nil {
		t.Fatal("expected error for missing client secret")
	}
}

func TestIGDB_Search_HappyPath(t *testing.T) {
	var lastBody string
	srv, tokenCalls := igdbTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("games: method = %s, want POST", r.Method)
		}
		if r.Header.Get("Client-ID") != "cid" || r.Header.Get("Authorization") != "Bearer tok123" {
			t.Errorf("games: missing auth headers: Client-ID=%q Authorization=%q", r.Header.Get("Client-ID"), r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		lastBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]igdbGame{
			{ID: 1, Name: "Elden Ring", Slug: "elden-ring", FirstReleaseDate: ip64(1645747200), AggregatedRating: fp(96), Rating: fp(90)},
			{ID: 2, Name: "Hollow Knight", Slug: "hollow-knight", FirstReleaseDate: ip64(1489968000), AggregatedRating: nil, Rating: fp(88)},
			{ID: 3, Name: "Unrated", Slug: "unrated", FirstReleaseDate: ip64(1489968000), AggregatedRating: nil, Rating: nil},
		})
	})

	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}

	year := 2022
	cands, err := client.Search(context.Background(), library.ScoreQuery{
		Name: "Elden Ring", MediaType: library.MediaTypeGame, ReleaseYear: &year,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(cands) != 2 { // "Unrated" has no score → filtered
		t.Fatalf("candidates = %d, want 2", len(cands))
	}
	// aggregated_rating is preferred and reported as metacritic.
	if cands[0].Title != "Elden Ring" || cands[0].Score != 96 || cands[0].Source != "metacritic" ||
		cands[0].Year != 2022 || cands[0].ExternalID != "1" || cands[0].URL != "https://www.igdb.com/games/elden-ring" {
		t.Fatalf("unexpected aggregated candidate: %+v", cands[0])
	}
	// community rating is the fallback and reported as igdb.
	if cands[1].Title != "Hollow Knight" || cands[1].Score != 88 || cands[1].Source != "igdb" ||
		cands[1].URL != "https://www.igdb.com/games/hollow-knight" {
		t.Fatalf("unexpected fallback candidate: %+v", cands[1])
	}
	if !strings.Contains(lastBody, `search "Elden Ring";`) || !strings.Contains(lastBody, "limit 10;") {
		t.Fatalf("query missing search/limit: %s", lastBody)
	}
	// 2022-01-01T00:00:00Z and 2023-01-01T00:00:00Z in unix seconds.
	if !strings.Contains(lastBody, "first_release_date >= 1640995200") || !strings.Contains(lastBody, "first_release_date < 1672531200") {
		t.Fatalf("query missing year filter: %s", lastBody)
	}

	// The token must be cached across searches.
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame}); err != nil {
		t.Fatalf("second Search: %v", err)
	}
	if tokenCalls.Load() != 1 {
		t.Fatalf("token requests = %d, want 1 (cached)", tokenCalls.Load())
	}
}

func TestIGDB_Search_EmptyResults(t *testing.T) {
	srv, _ := igdbTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})
	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "zzz", MediaType: library.MediaTypeGame})
	if err != nil || len(cands) != 0 {
		t.Fatalf("Search(empty) = %v (err %v), want empty", cands, err)
	}
}

func TestIGDB_Search_MalformedJSON(t *testing.T) {
	srv, _ := igdbTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}
	if _, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame}); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestIGDB_RefreshesTokenAfterUnauthorized(t *testing.T) {
	first := true
	srv, tokenCalls := igdbTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if first {
			first = false
			http.Error(w, `[{"title":"Unauthorized","status":401,"error":"Invalid access token"}]`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]igdbGame{{ID: 9, Name: "Ok", Slug: "ok", AggregatedRating: fp(80)}})
	})
	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame})
	if err != nil {
		t.Fatalf("Search after 401: %v", err)
	}
	if len(cands) != 1 || cands[0].Title != "Ok" {
		t.Fatalf("unexpected candidates after refresh: %+v", cands)
	}
	if tokenCalls.Load() != 2 {
		t.Fatalf("token requests = %d, want 2 (one refresh)", tokenCalls.Load())
	}
}

func TestIGDB_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"status":400,"message":"invalid client"}`, http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}
	_, err = client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame})
	if err == nil || !strings.Contains(err.Error(), "invalid client") {
		t.Fatalf("expected Twitch credential error, got %v", err)
	}
}

func TestIGDB_GameWithoutSlugUsesIDURL(t *testing.T) {
	srv, _ := igdbTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]igdbGame{{ID: 42, Name: "No Slug", AggregatedRating: fp(77)}})
	})
	client, err := NewIGDB(context.Background(), igdbProvider(srv))
	if err != nil {
		t.Fatalf("NewIGDB: %v", err)
	}
	cands, err := client.Search(context.Background(), library.ScoreQuery{Name: "x", MediaType: library.MediaTypeGame})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(cands) != 1 || cands[0].URL != "https://www.igdb.com/games/42" {
		t.Fatalf("unexpected candidate: %+v", cands)
	}
}
