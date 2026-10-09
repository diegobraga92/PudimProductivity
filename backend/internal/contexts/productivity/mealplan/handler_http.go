package mealplan

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("list meal plan items failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list meal plan items")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponses(items))
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		log.Error().Err(err).Msg("create meal plan item failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toResponses(items))
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Update(r.Context(), id, req.toInput())
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "meal plan item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("update meal plan item failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "meal plan item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("delete meal plan item failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete meal plan item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
