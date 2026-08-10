package mealplan

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateMealPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	in, err := req.toInput()
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "start_date/end_date/slot dates must be YYYY-MM-DD")
		return
	}
	plan, err := h.service.Create(r.Context(), in)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusCreated, toResponse(plan))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.List(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("list meal plans failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list meal plans")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponses(plans))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "planId")
	plan, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("plan_id", id).Msg("get meal plan failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get meal plan")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponse(plan))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "planId")
	var req CreateMealPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	in, err := req.toInput()
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "start_date/end_date/slot dates must be YYYY-MM-DD")
		return
	}
	plan, err := h.service.Update(r.Context(), id, in)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan not found")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponse(plan))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "planId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("plan_id", id).Msg("delete meal plan failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete meal plan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AssignSlot(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	slotID := chi.URLParam(r, "slotId")
	var req AssignSlotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	err := h.service.AssignSlot(r.Context(), planID, slotID, req.RecipeID, req.Notes)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan or slot not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("assign slot failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to assign slot")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GenerateShoppingList(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	items, err := h.service.GenerateShoppingList(r.Context(), planID)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("plan_id", planID).Msg("generate shopping list failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to generate shopping list")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toShoppingResponses(items))
}

func (h *Handler) GetShoppingList(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	items, err := h.service.GetShoppingList(r.Context(), planID)
	if err != nil {
		log.Error().Err(err).Str("plan_id", planID).Msg("get shopping list failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get shopping list")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toShoppingResponses(items))
}

// DownloadPDF streams a printable PDF of the meal plan (Phase 9b media
// processing). Used by the web "Download" button.
func (h *Handler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	data, err := h.service.RenderPDF(r.Context(), planID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			shared.WriteError(w, http.StatusNotFound, "meal plan not found")
			return
		}
		log.Error().Err(err).Str("plan_id", planID).Msg("render meal plan pdf failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to render PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="mealplan-%s.pdf"`, planID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Warn().Err(err).Str("plan_id", planID).Msg("stream meal plan pdf failed")
	}
}

func (h *Handler) ToggleShoppingItem(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	itemID := chi.URLParam(r, "itemId")
	err := h.service.ToggleShoppingItem(r.Context(), planID, itemID)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "shopping item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("toggle shopping item failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to toggle shopping item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planId")
	err := h.service.Publish(r.Context(), planID)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "meal plan not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("plan_id", planID).Msg("publish meal plan failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to publish meal plan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
