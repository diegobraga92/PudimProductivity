// Package googlebooks is the Google Books API adapter for the book-tracking
// module (Phase 5). It is the reference consumer of internal/httpclient: every
// lookup flows through the shared circuit breaker + rate limiter, so the
// backend fails fast when Google is unreachable and never blows a quota.
package googlebooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/diegobraga92/pudimproductivity/backend/internal/httpclient"
)

// BookInfo is the flattened, API-agnostic book metadata the module stores.
type BookInfo struct {
	ISBN          string
	Title         string
	Authors       []string
	Publisher     string
	PublishedDate string
	Description   string
	PageCount     int
	ThumbnailURL  string
}

// ErrNotFound is returned when Google Books has no volume for the ISBN.
var ErrNotFound = errors.New("googlebooks: no volume found for ISBN")

// Config tunes the adapter. Zero values fall back to httpclient defaults.
type Config struct {
	// APIKey is the optional Google Books API key (recommended; ups quota).
	APIKey string
	// BaseURL overrides the API base (tests use a stub server).
	BaseURL string
	// HTTP applies the shared client policy (retries, rate limit, breaker).
	HTTP httpclient.Config
}

const defaultBaseURL = "https://www.googleapis.com/books/v1"

type Client struct {
	http   *httpclient.Client
	apiKey string
	base   string
}

func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	if cfg.HTTP.Rate == 0 {
		cfg.HTTP.Rate = 10 // stay comfortably inside Google's quota
		cfg.HTTP.Burst = 10
	}
	return &Client{http: httpclient.New(cfg.HTTP), apiKey: cfg.APIKey, base: strings.TrimSuffix(base, "/")}
}

// LookupByISBN fetches metadata for a normalized ISBN. Returns ErrNotFound
// when no volume matches.
func (c *Client) LookupByISBN(ctx context.Context, isbn string) (*BookInfo, error) {
	u, err := url.Parse(c.base + "/volumes")
	if err != nil {
		return nil, fmt.Errorf("googlebooks: parse base URL: %w", err)
	}
	q := u.Query()
	q.Set("q", "isbn:"+isbn)
	if c.apiKey != "" {
		q.Set("key", c.apiKey)
	}
	u.RawQuery = q.Encode()

	body, err := c.http.Get(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("googlebooks: lookup %s: %w", isbn, err)
	}

	var resp volumesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("googlebooks: decode response: %w", err)
	}
	if resp.TotalItems == 0 || len(resp.Items) == 0 {
		return nil, ErrNotFound
	}
	return resp.Items[0].toBookInfo(isbn), nil
}

// --- Google Books response shapes (only the fields we store) ---

type volumesResponse struct {
	TotalItems int           `json:"totalItems"`
	Items      []volumeEntry `json:"items"`
}

type volumeEntry struct {
	VolumeInfo struct {
		Title         string   `json:"title"`
		Authors       []string `json:"authors"`
		Publisher     string   `json:"publisher"`
		PublishedDate string   `json:"publishedDate"`
		Description   string   `json:"description"`
		PageCount     int      `json:"pageCount"`
		ImageLinks    struct {
			Thumbnail string `json:"thumbnail"`
		} `json:"imageLinks"`
	} `json:"volumeInfo"`
}

func (v volumeEntry) toBookInfo(isbn string) *BookInfo {
	return &BookInfo{
		ISBN:          isbn,
		Title:         v.VolumeInfo.Title,
		Authors:       v.VolumeInfo.Authors,
		Publisher:     v.VolumeInfo.Publisher,
		PublishedDate: v.VolumeInfo.PublishedDate,
		Description:   v.VolumeInfo.Description,
		PageCount:     v.VolumeInfo.PageCount,
		ThumbnailURL:  v.VolumeInfo.ImageLinks.Thumbnail,
	}
}
