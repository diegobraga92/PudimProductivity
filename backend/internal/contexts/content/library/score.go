package library

import "context"

// TODO: Make it on by default.

// ScoreLookupFeatureFlag gates the score-lookup feature (off by default).
const ScoreLookupFeatureFlag = "library.score_lookup_enabled"

// ScoreQuery identifies the media whose rating should be searched.
type ScoreQuery struct {
	Name        string
	MediaType   MediaType
	ReleaseYear *int
}

// ScoreCandidate is a single match returned by a rating provider (OMDb, RAWG,
// ...). Score is always on the 0-100 scale shared by the library_items table.
type ScoreCandidate struct {
	Title      string
	Year       int
	Score      float64
	Source     string
	ExternalID string
	URL        string
}

// ScoreLookupClient searches external rating providers.
type ScoreLookupClient interface {
	Search(ctx context.Context, query ScoreQuery) ([]ScoreCandidate, error)
}

// ScoreLookupProvider extends ScoreLookupClient with a Configured query so
// handlers can distinguish "not configured / degraded" (503) from "no ratings
// found" (200 with an empty list).
type ScoreLookupProvider interface {
	ScoreLookupClient
	Configured() bool
}

// NoopScoreLookup is the default client when no provider is configured or the
// feature flag is disabled.
type NoopScoreLookup struct{}

func (NoopScoreLookup) Search(context.Context, ScoreQuery) ([]ScoreCandidate, error) {
	return nil, nil
}

// Configured reports whether the client can serve lookups.
func (NoopScoreLookup) Configured() bool { return false }
