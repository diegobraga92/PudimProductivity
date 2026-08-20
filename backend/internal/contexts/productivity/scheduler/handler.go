package scheduler

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
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
			httpx.WriteError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
			return
		}
		date = parsed
	}

	suggestion, err := h.service.SuggestDay(r.Context(), date)
	if err != nil {
		log.Error().Err(err).Msg("suggest day failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to build schedule")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, suggestion)
}
