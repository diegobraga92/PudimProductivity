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

// defaultOMDBBaseURL is the official OMDb API endpoint.
const defaultOMDBBaseURL = "https://www.omdbapi.com"

// maxOMDBDetails caps how many per-title detail requests a single search
// issues, keeping us within OMDb's free-tier quota (1,000 requests/day).
const maxOMDBDetails = 3

// omdbSearchResponse mirrors the fields we consume from OMDb's search response.
type omdbSearchResponse struct {
	Search []struct {
		Title  string `json:"Title"`
		Year   string `json:"Year"`
		ImdbID string `json:"imdbID"`
	} `json:"Search"`
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

// omdbDetailResponse mirrors the fields we consume from OMDb's detail response.
// Metascore is OMDb's copy of the Metacritic 0-100 score ("N/A" when absent).
type omdbDetailResponse struct {
	ImdbRating string `json:"imdbRating"`
	Metascore  string `json:"Metascore"`
	Response   string `json:"Response"`
	Error      string `json:"Error"`
}

// omdbRating is a resolved rating for a title, on the source's native scale.
type omdbRating struct {
	score  float64
	source string
}

// preferred returns the most relevant rating for the feature.
func (d omdbDetailResponse) preferred() *omdbRating {
	if ms, ok := parseOMDBScore(d.Metascore); ok {
		return &omdbRating{score: ms, source: string(library.ScoreSourceMetacritic)}
	}
	if ir, ok := parseOMDBScore(d.ImdbRating); ok {
		return &omdbRating{score: ir, source: string(library.ScoreSourceIMDb)}
	}
	return nil
}

// parseOMDBScore parses an OMDb numeric rating, treating "N/A" and zero as "not rated yet".
func parseOMDBScore(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return v, true
}

// omdbClient searches OMDb for films/series and surfaces the Metacritic score
// when available (falling back to the IMDb rating).
type omdbClient struct {
	hc   *httpclient.Client
	key  string
	base string
}

// NewOMDB builds the OMDb adapter (films/series → Metacritic score, with an
// IMDb-rating fallback).
func NewOMDB(_ context.Context, cfg ProviderConfig) (library.ScoreLookupClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("omdb: API key required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultOMDBBaseURL
	}
	return &omdbClient{
		hc:   httpclient.New(httpclient.Config{Retries: 1, Rate: 2, Burst: 5}),
		key:  cfg.APIKey,
		base: strings.TrimRight(base, "/"),
	}, nil
}

func (o *omdbClient) Search(ctx context.Context, query library.ScoreQuery) ([]library.ScoreCandidate, error) {
	params := url.Values{}
	params.Set("apikey", o.key)
	params.Set("s", query.Name)
	params.Set("type", omdbType(query.MediaType))
	if query.ReleaseYear != nil {
		params.Set("y", strconv.Itoa(*query.ReleaseYear))
	}
	body, err := o.hc.Get(ctx, o.base+"?"+params.Encode())
	if err != nil {
		return nil, o.mapErr(err)
	}
	var search omdbSearchResponse
	if err := json.Unmarshal(body, &search); err != nil {
		return nil, fmt.Errorf("omdb: parse search response: %w", err)
	}
	if search.Response == "False" {
		if strings.Contains(strings.ToLower(search.Error), "not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("omdb: %s", search.Error)
	}

	cands := make([]library.ScoreCandidate, 0, len(search.Search))
	for i, hit := range search.Search {
		if i >= maxOMDBDetails {
			break
		}
		detail, err := o.fetchDetail(ctx, hit.ImdbID)
		if err != nil {
			return nil, err
		}
		rating := detail.preferred()
		if rating == nil {
			continue // no Metacritic or IMDb rating yet — skip the candidate
		}
		cands = append(cands, library.ScoreCandidate{
			Title:      hit.Title,
			Year:       omdbYear(hit.Year),
			Score:      rating.score,
			Source:     rating.source,
			ExternalID: hit.ImdbID,
			URL:        "https://www.imdb.com/title/" + hit.ImdbID,
		})
	}
	return cands, nil
}

// fetchDetail returns the full rating payload for a title.
func (o *omdbClient) fetchDetail(ctx context.Context, imdbID string) (omdbDetailResponse, error) {
	params := url.Values{}
	params.Set("apikey", o.key)
	params.Set("i", imdbID)
	body, err := o.hc.Get(ctx, o.base+"?"+params.Encode())
	if err != nil {
		return omdbDetailResponse{}, o.mapErr(err)
	}
	var detail omdbDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return omdbDetailResponse{}, fmt.Errorf("omdb: parse detail response: %w", err)
	}
	if detail.Response == "False" {
		return omdbDetailResponse{}, fmt.Errorf("omdb: %s", detail.Error)
	}
	return detail, nil
}

func omdbType(mt library.MediaType) string {
	if mt == library.MediaTypeSeries {
		return "series"
	}
	return "movie"
}

// omdbYear extracts the leading year from OMDb's "1999" / "1999–2000" values.
func omdbYear(s string) int {
	if i := strings.IndexAny(s, "–-"); i >= 0 {
		s = s[:i]
	}
	y, _ := strconv.Atoi(strings.TrimSpace(s))
	return y
}

func (o *omdbClient) mapErr(err error) error {
	var se *httpclient.StatusError
	if errors.As(err, &se) && se.Status == http.StatusUnauthorized {
		return fmt.Errorf("omdb: invalid API key")
	}
	return fmt.Errorf("omdb: %w", err)
}
