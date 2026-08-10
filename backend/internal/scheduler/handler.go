package scheduler

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Service is the handler-level interface.
type Service interface {
	SuggestDay(ctx context.Context, date time.Time) (*Suggestion, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetSchedule returns the suggested plan for ?date=YYYY-MM-DD (defaults to
// today).
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	date := time.Now()
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
			return
		}
		date = parsed
	}

	suggestion, err := h.service.SuggestDay(r.Context(), date)
	if err != nil {
		log.Error().Err(err).Msg("suggest day failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to build schedule")
		return
	}
	shared.WriteJSON(w, http.StatusOK, suggestion)
}
