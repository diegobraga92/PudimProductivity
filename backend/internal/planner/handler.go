package planner

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service *PlannerService
}

func NewHandler(service *PlannerService) *Handler {
	return &Handler{service: service}
}

// DTO
type plannerEntryResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Days      []string `json:"days"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Color     string   `json:"color"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type createPlannerEntryRequest struct {
	Title     string   `json:"title"`
	Days      []string `json:"days"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Color     string   `json:"color"`
}

type updatePlannerEntryRequest struct {
	Title     *string   `json:"title"`
	Days      *[]string `json:"days"`
	StartTime *string   `json:"start_time"`
	EndTime   *string   `json:"end_time"`
	Color     *string   `json:"color"`
}

func toPlannerEntryResponse(e *PlannerEntry) plannerEntryResponse {
	return plannerEntryResponse{
		ID:        e.ID,
		Title:     e.Title,
		Days:      e.Days,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		Color:     e.Color,
		CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: e.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// GET /api/v1/planner
func (h *Handler) ListPlannerEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.ListPlannerEntries(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list planner entries")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list planner entries")
		return
	}

	responses := make([]plannerEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = toPlannerEntryResponse(e)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}

// POST /api/v1/planner
func (h *Handler) CreatePlannerEntry(w http.ResponseWriter, r *http.Request) {
	var req createPlannerEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		shared.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(req.Days) == 0 {
		shared.WriteError(w, http.StatusBadRequest, "days is required")
		return
	}
	if req.StartTime == "" {
		shared.WriteError(w, http.StatusBadRequest, "start_time is required")
		return
	}
	if req.EndTime == "" {
		shared.WriteError(w, http.StatusBadRequest, "end_time is required")
		return
	}

	entry, err := h.service.CreatePlannerEntry(r.Context(), req.Title, req.Days, req.StartTime, req.EndTime, req.Color)
	if err != nil {
		log.Error().Err(err).Msg("failed to create planner entry")
		shared.WriteError(w, http.StatusInternalServerError, "failed to create planner entry")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, toPlannerEntryResponse(entry))
}

// GET /api/v1/planner/{entryId}
func (h *Handler) GetPlannerEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "entryId")

	entry, err := h.service.GetPlannerEntry(r.Context(), entryID)
	if err != nil {
		if errors.Is(err, ErrPlannerEntryNotFound) {
			shared.WriteError(w, http.StatusNotFound, "planner entry not found")
			return
		}
		log.Error().Err(err).Str("entry_id", entryID).Msg("failed to get planner entry")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get planner entry")
		return
	}

	shared.WriteJSON(w, http.StatusOK, toPlannerEntryResponse(entry))
}

// PUT /api/v1/planner/{entryId}
func (h *Handler) UpdatePlannerEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "entryId")

	var req updatePlannerEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entry, err := h.service.UpdatePlannerEntry(r.Context(), entryID, req.Title, req.Days, req.StartTime, req.EndTime, req.Color)
	if err != nil {
		if errors.Is(err, ErrPlannerEntryNotFound) {
			shared.WriteError(w, http.StatusNotFound, "planner entry not found")
			return
		}
		log.Error().Err(err).Str("entry_id", entryID).Msg("failed to update planner entry")
		shared.WriteError(w, http.StatusInternalServerError, "failed to update planner entry")
		return
	}

	shared.WriteJSON(w, http.StatusOK, toPlannerEntryResponse(entry))
}

// DELETE /api/v1/planner/{entryId}
func (h *Handler) DeletePlannerEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "entryId")

	if err := h.service.DeletePlannerEntry(r.Context(), entryID); err != nil {
		if errors.Is(err, ErrPlannerEntryNotFound) {
			shared.WriteError(w, http.StatusNotFound, "planner entry not found")
			return
		}
		log.Error().Err(err).Str("entry_id", entryID).Msg("failed to delete planner entry")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete planner entry")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
