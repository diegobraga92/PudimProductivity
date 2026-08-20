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

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/content/library"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/httpclient"
)

// defaultRAWGBaseURL is the official RAWG API endpoint.
const defaultRAWGBaseURL = "https://api.rawg.io/api"

// rawgResponse mirrors the fields we consume from RAWG's /games search
// response. Metacritic ratings are already present in the search results.
type rawgResponse struct {
	Results []struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Released   string `json:"released"`
		Metacritic *int   `json:"metacritic"`
		Slug       string `json:"slug"`
	} `json:"results"`
}

// rawgClient searches RAWG for games and surfaces the Metacritic rating.
type rawgClient struct {
	hc   *httpclient.Client
	key  string
	base string
}

// NewRAWG builds the RAWG adapter (games → Metacritic rating).
func NewRAWG(_ context.Context, cfg ProviderConfig) (library.ScoreLookupClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("rawg: API key required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultRAWGBaseURL
	}
	return &rawgClient{
		hc:   httpclient.New(httpclient.Config{Retries: 1, Rate: 2, Burst: 5}),
		key:  cfg.APIKey,
		base: strings.TrimRight(base, "/"),
	}, nil
}

func (r *rawgClient) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	params := url.Values{}
	params.Set("key", r.key)
	params.Set("search", query.Name)
	params.Set("page_size", "5")
	if query.ReleaseYear != nil {
		params.Set("dates", fmt.Sprintf("%d-01-01,%d-12-31", *query.ReleaseYear, *query.ReleaseYear))
	}
	body, err := r.hc.Get(ctx, r.base+"/games?"+params.Encode())
	if err != nil {
		return nil, r.mapErr(err)
	}
	var resp rawgResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("rawg: parse response: %w", err)
	}

	cands := make([]library.ScoreCandidate, 0, len(resp.Results))
	for _, g := range resp.Results {
		if g.Metacritic == nil {
			continue // no Metacritic rating for this game
		}
		cands = append(cands, library.ScoreCandidate{
			Title:      g.Name,
			Year:       rawgYear(g.Released),
			Score:      float64(*g.Metacritic),
			Source:     string(library.ScoreSourceMetacritic),
			ExternalID: strconv.Itoa(g.ID),
			URL:        "https://rawg.io/games/" + g.Slug,
		})
	}
	return cands, nil
}

// rawgYear extracts the leading year from RAWG's "2022-02-25" values.
func rawgYear(released string) int {
	if len(released) >= 4 {
		if y, err := strconv.Atoi(released[:4]); err == nil {
			return y
		}
	}
	return 0
}

func (r *rawgClient) mapErr(err error) error {
	var se *httpclient.StatusError
	if errors.As(err, &se) && se.Status == http.StatusUnauthorized {
		return fmt.Errorf("rawg: invalid API key")
	}
	return fmt.Errorf("rawg: %w", err)
}
