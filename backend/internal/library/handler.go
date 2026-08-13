package library

import (
	"context"
	"time"
)

// --- Requests ---

type CreateItemRequest struct {
	Name        string `json:"name"`
	MediaType   string `json:"media_type"`
	ReleaseYear *int   `json:"release_year"`
	Done        bool   `json:"done"`
	Notes       string `json:"notes"`
}

func (req CreateItemRequest) toInput() CreateInput {
	return CreateInput{
		Name:        req.Name,
		MediaType:   MediaType(req.MediaType),
		ReleaseYear: req.ReleaseYear,
		Done:        req.Done,
		Notes:       req.Notes,
	}
}

type UpdateItemRequest struct {
	Name        *string `json:"name"`
	MediaType   *string `json:"media_type"`
	ReleaseYear **int   `json:"release_year"`
	Done        *bool   `json:"done"`
	Notes       *string `json:"notes"`
}

func (req UpdateItemRequest) toInput() UpdateInput {
	in := UpdateInput{
		Name:        req.Name,
		ReleaseYear: req.ReleaseYear,
		Done:        req.Done,
		Notes:       req.Notes,
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	MediaType   string `json:"media_type"`
	ReleaseYear *int   `json:"release_year"`
	Done        bool   `json:"done"`
	Notes       string `json:"notes"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toResponse(it *Item) ItemResponse {
	return ItemResponse{
		ID:          it.ID,
		Name:        it.Name,
		MediaType:   string(it.MediaType),
		ReleaseYear: it.ReleaseYear,
		Done:        it.Done,
		Notes:       it.Notes,
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

// --- Service interface (consumer-side, handler level) ---

type Service interface {
	Create(ctx context.Context, in CreateInput) (*Item, error)
	Import(ctx context.Context, in []CreateInput) (*ImportResult, error)
	Get(ctx context.Context, id string) (*Item, error)
	List(ctx context.Context, mediaType string, done *bool) ([]*Item, error)
	Update(ctx context.Context, id string, in UpdateInput) (*Item, error)
	Delete(ctx context.Context, id string) error
}
