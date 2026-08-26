package scoringapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/httpclient"
)

// Default IGDB / Twitch endpoints.
const (
	defaultIGDBAuthURL = "https://id.twitch.tv/oauth2/token"
	defaultIGDBBaseURL = "https://api.igdb.com/v4"
)

// igdbClientSecretSetting is the score_providers.settings key holding the Twitch
// client secret (the client ID itself is stored in the provider API key).
const igdbClientSecretSetting = "client_secret"

// igdbTokenURLSetting optionally overrides the OAuth token endpoint (tests,
// self-hosted proxies). It is accepted from the settings bag but not exposed in
// the admin UI.
const igdbTokenURLSetting = "token_url"

// maxIGDBCandidates caps how many games a single search returns.
const maxIGDBCandidates = 10

// igdbTokenResponse mirrors Twitch's OAuth client-credentials response. The
// Status/Message fields are present in error payloads too.
type igdbTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Status      int    `json:"status"`
	Message     string `json:"message"`
}

// igdbAPIError mirrors a single entry of IGDB's JSON error array.
type igdbAPIError struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// igdbGame mirrors the games fields we consume. aggregated_rating is the 0-100
// score compiled from Metacritic and OpenCritic.
type igdbGame struct {
	ID                    int      `json:"id"`
	Name                  string   `json:"name"`
	Slug                  string   `json:"slug"`
	FirstReleaseDate      *int64   `json:"first_release_date"`
	AggregatedRating      *float64 `json:"aggregated_rating"`
	AggregatedRatingCount int      `json:"aggregated_rating_count"`
	Rating                *float64 `json:"rating"`
}

// igdbClient searches IGDB (via Twitch OAuth) for games and surfaces the
// aggregated (Metacritic/OpenCritic) rating, falling back to the community
// rating when no aggregated score exists.
type igdbClient struct {
	hc           *httpclient.Client
	clientID     string
	clientSecret string
	apiBase      string
	tokenURL     string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

// NewIGDB builds the IGDB adapter (games → aggregated rating).
func NewIGDB(_ context.Context, cfg ProviderConfig) (library.ScoreLookupClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("igdb: Twitch client ID (API key) required")
	}
	secret := cfg.Settings[igdbClientSecretSetting]
	if secret == "" {
		return nil, fmt.Errorf("igdb: Twitch client secret required (provider setting %q)", igdbClientSecretSetting)
	}
	apiBase := cfg.BaseURL
	if apiBase == "" {
		apiBase = defaultIGDBBaseURL
	}
	tokenURL := cfg.Settings[igdbTokenURLSetting]
	if tokenURL == "" {
		tokenURL = defaultIGDBAuthURL
	}
	return &igdbClient{
		hc:           httpclient.New(httpclient.Config{Retries: 1, Rate: 2, Burst: 5}),
		clientID:     cfg.APIKey,
		clientSecret: secret,
		apiBase:      strings.TrimRight(apiBase, "/"),
		tokenURL:     tokenURL,
	}, nil
}

func (g *igdbClient) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	queryBody := igdbQuery(query)
	resp, err := g.searchGames(ctx, queryBody)
	if err != nil {
		// A stale or revoked token is worth exactly one refresh-and-retry.
		if invalidIGDBCredentials(err) {
			g.dropToken()
			resp, err = g.searchGames(ctx, queryBody)
		}
		if err != nil {
			return nil, g.mapErr(err)
		}
	}

	var games []igdbGame
	if err := json.Unmarshal(resp, &games); err != nil {
		return nil, fmt.Errorf("igdb: parse games response: %w", err)
	}

	cands := make([]library.ScoreCandidate, 0, len(games))
	for _, game := range games {
		score, source := igdbScore(game)
		if score == nil {
			continue // no rating yet — skip the candidate
		}
		gameURL := "https://www.igdb.com/games/" + strconv.Itoa(game.ID)
		if game.Slug != "" {
			gameURL = "https://www.igdb.com/games/" + game.Slug
		}
		cands = append(cands, library.ScoreCandidate{
			Title:      game.Name,
			Year:       igdbYear(game.FirstReleaseDate),
			Score:      *score,
			Source:     source,
			ExternalID: strconv.Itoa(game.ID),
			URL:        gameURL,
		})
	}
	return cands, nil
}

// searchGames runs a single IGDB /games query with a valid Bearer token.
func (g *igdbClient) searchGames(ctx context.Context, query string) ([]byte, error) {
	token, err := g.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Client-ID":     g.clientID,
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}
	return g.hc.DoWith(ctx, http.MethodPost, g.apiBase+"/games", "text/plain", []byte(query), headers)
}

// accessToken returns a cached, unexpired Twitch app token, fetching a new one
// when needed. Safe for concurrent use.
func (g *igdbClient) accessToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.token != "" && time.Now().Before(g.expiresAt) {
		return g.token, nil
	}

	form := url.Values{}
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.clientSecret)
	form.Set("grant_type", "client_credentials")
	body, err := g.hc.Post(ctx, g.tokenURL, "application/x-www-form-urlencoded", []byte(form.Encode()))
	if err != nil {
		var se *httpclient.StatusError
		if errors.As(err, &se) {
			var terr igdbTokenResponse
			if json.Unmarshal(se.Body, &terr) == nil && terr.Message != "" {
				return "", fmt.Errorf("igdb: token request failed: %s", terr.Message)
			}
			if se.Status == http.StatusBadRequest {
				return "", fmt.Errorf("igdb: invalid Twitch client ID or secret")
			}
		}
		return "", g.mapErr(err)
	}
	var resp igdbTokenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("igdb: parse token response: %w", err)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("igdb: token response contained no access token")
	}

	g.token = resp.AccessToken
	// Refresh 5 minutes before expiry; Twitch app tokens last ~60 days.
	ttl := time.Duration(resp.ExpiresIn) * time.Second
	if ttl < time.Minute {
		ttl = time.Minute
	}
	g.expiresAt = time.Now().Add(ttl - 5*time.Minute)
	return g.token, nil
}

func (g *igdbClient) dropToken() {
	g.mu.Lock()
	g.token = ""
	g.mu.Unlock()
}

// igdbQuery builds the Apicalypse query for a name search.
func igdbQuery(query library.ScoreQuery) string {
	var b strings.Builder
	b.WriteString("fields name,slug,first_release_date,aggregated_rating,aggregated_rating_count,rating,rating_count;")
	if query.ReleaseYear != nil {
		start := time.Date(*query.ReleaseYear, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		end := time.Date(*query.ReleaseYear+1, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		fmt.Fprintf(&b, "where first_release_date >= %d & first_release_date < %d;", start, end)
	}
	fmt.Fprintf(&b, "search \"%s\";limit %d;", strings.ReplaceAll(query.Name, `"`, `\"`), maxIGDBCandidates)
	return b.String()
}

// igdbScore returns the most relevant score for a game.
func igdbScore(game igdbGame) (*float64, string) {
	if game.AggregatedRating != nil && *game.AggregatedRating > 0 {
		return game.AggregatedRating, string(library.ScoreSourceMetacritic)
	}
	if game.Rating != nil && *game.Rating > 0 {
		return game.Rating, string(library.ScoreSourceIGDB)
	}
	return nil, ""
}

// igdbYear extracts the release year from IGDB's unix-seconds timestamp.
func igdbYear(unix *int64) int {
	if unix == nil || *unix <= 0 {
		return 0
	}
	return time.Unix(*unix, 0).UTC().Year()
}

// invalidIGDBCredentials reports whether the error looks like a rejected or
// expired token (HTTP 401 from IGDB).
func invalidIGDBCredentials(err error) bool {
	var se *httpclient.StatusError
	return errors.As(err, &se) && se.Status == http.StatusUnauthorized
}

func (g *igdbClient) mapErr(err error) error {
	var se *httpclient.StatusError
	if errors.As(err, &se) {
		switch se.Status {
		case http.StatusUnauthorized:
			return fmt.Errorf("igdb: invalid Twitch client ID or secret")
		case http.StatusTooManyRequests:
			return fmt.Errorf("igdb: rate limited by Twitch/IGDB")
		}
		if len(se.Body) > 0 {
			var apiErrs []igdbAPIError
			if json.Unmarshal(se.Body, &apiErrs) == nil && len(apiErrs) > 0 && apiErrs[0].Error != "" {
				return fmt.Errorf("igdb: %s", apiErrs[0].Error)
			}
		}
	}
	return fmt.Errorf("igdb: %w", err)
}
