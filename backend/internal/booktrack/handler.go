package booktrack

import (
	"context"
	"time"
)

// --- Requests ---

type AddByISBNRequest struct {
	ISBN string `json:"isbn"`
}

type AddManualRequest struct {
	ISBN          string   `json:"isbn"`
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	Publisher     string   `json:"publisher"`
	PublishedDate string   `json:"published_date"`
	Description   string   `json:"description"`
	PageCount     int      `json:"page_count"`
	ThumbnailURL  string   `json:"thumbnail_url"`
	Status        string   `json:"status"`
}

func (req AddManualRequest) toInput() AddInput {
	return AddInput{
		ISBN:          req.ISBN,
		Title:         req.Title,
		Authors:       req.Authors,
		Publisher:     req.Publisher,
		PublishedDate: req.PublishedDate,
		Description:   req.Description,
		PageCount:     req.PageCount,
		ThumbnailURL:  req.ThumbnailURL,
		Status:        BookStatus(req.Status),
	}
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// --- Responses ---

type BookResponse struct {
	ID            string   `json:"id"`
	ISBN          string   `json:"isbn"`
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	Publisher     string   `json:"publisher,omitempty"`
	PublishedDate string   `json:"published_date,omitempty"`
	Description   string   `json:"description,omitempty"`
	PageCount     int      `json:"page_count"`
	ThumbnailURL  string   `json:"thumbnail_url,omitempty"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func toResponse(b *Book) BookResponse {
	return BookResponse{
		ID:            b.ID,
		ISBN:          b.ISBN,
		Title:         b.Title,
		Authors:       b.Authors,
		Publisher:     b.Publisher,
		PublishedDate: b.PublishedDate,
		Description:   b.Description,
		PageCount:     b.PageCount,
		ThumbnailURL:  b.ThumbnailURL,
		Status:        string(b.Status),
		CreatedAt:     b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     b.UpdatedAt.Format(time.RFC3339),
	}
}

func toResponses(books []*Book) []BookResponse {
	out := make([]BookResponse, 0, len(books))
	for _, b := range books {
		out = append(out, toResponse(b))
	}
	return out
}

// --- Service interface (consumer-side, handler level) ---

type Service interface {
	AddByISBN(ctx context.Context, isbn string) (*Book, error)
	AddManual(ctx context.Context, in AddInput) (*Book, error)
	Get(ctx context.Context, id string) (*Book, error)
	List(ctx context.Context, status string) ([]*Book, error)
	UpdateStatus(ctx context.Context, id string, status BookStatus) error
	Delete(ctx context.Context, id string) error
}
