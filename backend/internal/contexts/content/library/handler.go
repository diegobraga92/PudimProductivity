package library

import (
	"context"
	"time"
)

// --- Requests ---

type CreateItemRequest struct {
	Name        string   `json:"name"`
	MediaType   string   `json:"media_type"`
	ReleaseYear *int     `json:"release_year"`
	Done        bool     `json:"done"`
	Notes       string   `json:"notes"`
	Subtype     string   `json:"subtype"`
	Score       *float64 `json:"score"`
	ScoreSource string   `json:"score_source"`
}

func (req CreateItemRequest) toInput() CreateInput {
	return CreateInput{
		Name:        req.Name,
		MediaType:   MediaType(req.MediaType),
		ReleaseYear: req.ReleaseYear,
		Done:        req.Done,
		Notes:       req.Notes,
		Subtype:     req.Subtype,
		Score:       req.Score,
		ScoreSource: req.ScoreSource,
	}
}

type UpdateItemRequest struct {
	Name        *string   `json:"name"`
	MediaType   *string   `json:"media_type"`
	ReleaseYear **int     `json:"release_year"`
	Done        *bool     `json:"done"`
	Notes       *string   `json:"notes"`
	Subtype     *string   `json:"subtype"`
	Score       **float64 `json:"score"`
	ScoreSource *string   `json:"score_source"`
}

func (req UpdateItemRequest) toInput() UpdateInput {
	in := UpdateInput{
		Name:        req.Name,
		ReleaseYear: req.ReleaseYear,
		Done:        req.Done,
		Notes:       req.Notes,
		Subtype:     req.Subtype,
		Score:       req.Score,
		ScoreSource: req.ScoreSource,
	}
	if req.MediaType != nil {
		mt := MediaType(*req.MediaType)
		in.MediaType = &mt
	}
	return in
}

type ImportItemsRequest struct {
	Items []CreateItemRequest `json:"items"`
}

// --- Responses ---

type ItemResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	MediaType   string   `json:"media_type"`
	ReleaseYear *int     `json:"release_year"`
	Done        bool     `json:"done"`
	Notes       string   `json:"notes"`
	Subtype     string   `json:"subtype"`
	Score       *float64 `json:"score"`
	ScoreSource string   `json:"score_source"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func toResponse(it *Item) ItemResponse {
	return ItemResponse{
		ID:          it.ID,
		Name:        it.Name,
		MediaType:   string(it.MediaType),
		ReleaseYear: it.ReleaseYear,
		Done:        it.Done,
		Notes:       it.Notes,
		Subtype:     it.Subtype,
		Score:       it.Score,
		ScoreSource: it.ScoreSource,
		CreatedAt:   it.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   it.UpdatedAt.Format(time.RFC3339),
	}
}

func toResponses(items []*Item) []ItemResponse {
	out := make([]ItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toResponse(it))
	}
	return out
}

// ScoreCandidateResponse is a single rating-provider match for the score
// search endpoint.
type ScoreCandidateResponse struct {
	Title      string  `json:"title"`
	Year       int     `json:"year"`
	Score      float64 `json:"score"`
	Source     string  `json:"score_source"`
	ExternalID string  `json:"external_id"`
	URL        string  `json:"url"`
}

func toScoreResponses(cands []ScoreCandidate) []ScoreCandidateResponse {
	out := make([]ScoreCandidateResponse, 0, len(cands))
	for _, c := range cands {
		// ScoreCandidateResponse shares ScoreCandidate's fields 1:1 (the JSON
		// tag renames Source → score_source), so a plain conversion suffices.
		out = append(out, ScoreCandidateResponse(c))
	}
	return out
}

// ScoreBatchItemRequest is a single title to look up in a batch.
type ScoreBatchItemRequest struct {
	Name      string `json:"name"`
	MediaType string `json:"type"`
	Year      *int   `json:"year"`
}

// ScoreBatchRequest is the payload for POST /library/score/batch.
type ScoreBatchRequest struct {
	Items []ScoreBatchItemRequest `json:"items"`
}

// ScoreBatchResult is the outcome for one requested item. Index echoes the
// input order so clients can match results to rows. Error is present when the
// provider (or the request row) failed; otherwise candidates holds the matches.
type ScoreBatchResult struct {
	Index      int                      `json:"index"`
	Candidates []ScoreCandidateResponse `json:"candidates"`
	Error      string                   `json:"error,omitempty"`
}

// ScoreBatchResponse is the result of a batch lookup.
type ScoreBatchResponse struct {
	Results []ScoreBatchResult `json:"results"`
}

// --- Service interface (consumer-side, handler level) ---

type Service interface {
	Create(ctx context.Context, in CreateInput) (*Item, error)
	Import(ctx context.Context, in []CreateInput) (*ImportResult, error)
	Get(ctx context.Context, id string) (*Item, error)
	List(ctx context.Context, filter ListFilter) ([]*Item, error)
	DistinctSubtypes(ctx context.Context, mediaType string) ([]string, error)
	Update(ctx context.Context, id string, in UpdateInput) (*Item, error)
	Delete(ctx context.Context, id string) error
}
