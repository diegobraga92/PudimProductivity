package library

import "context"

// ScoreLookupFeatureFlag gates the score-lookup feature (off by default). See
// docs/adr/007-external-api-integrations.md for the graceful-degradation model.
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

// ScoreLookupClient searches external rating providers. Consumers depend on
// this interface (per ADR 007) so production can inject the configured
// composite and tests can inject stubs.
type ScoreLookupClient interface {
	Search(ctx context.Context, query ScoreQuery) ([]ScoreCandidate, error)
}

// ScoreLookupProvider extends ScoreLookupClient with a Configured query so
// handlers can distinguish "not configured / degraded" (503) from "no ratings
// found" (200 with an empty list). Kept as a separate interface so plain
// ScoreLookupClient stubs in tests do not have to implement it.
type ScoreLookupProvider interface {
	ScoreLookupClient
	Configured() bool
}

// NoopScoreLookup is the default client when no provider is configured or the
// feature flag is disabled. It always returns no candidates — the handler maps
// this to a 503 so clients can tell "not configured" from "no ratings found".
type NoopScoreLookup struct{}

func (NoopScoreLookup) Search(context.Context, ScoreQuery) ([]ScoreCandidate, error) {
	return nil, nil
}

// Configured reports whether the client can serve lookups. Noop is never
// configured; a concrete client with live providers is.
func (NoopScoreLookup) Configured() bool { return false }
