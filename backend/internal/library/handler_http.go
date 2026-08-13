package library

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// maxImportItems caps a single bulk-import request to keep the payload sane.
const maxImportItems = 5000

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		log.Error().Err(err).Msg("create library item failed")
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusCreated, toResponse(item))
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		shared.WriteError(w, http.StatusBadRequest, "no items to import")
		return
	}
	if len(req.Items) > maxImportItems {
		shared.WriteError(w, http.StatusBadRequest, "too many items in a single import")
		return
	}

	inputs := make([]CreateInput, 0, len(req.Items))
	for _, it := range req.Items {
		inputs = append(inputs, it.toInput())
	}
	result, err := h.service.Import(r.Context(), inputs)
	if err != nil {
		log.Error().Err(err).Msg("import library items failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to import library items")
		return
	}
	shared.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	var done *bool
	if raw := r.URL.Query().Get("done"); raw != "" {
		switch raw {
		case "true", "1", "yes":
			v := true
			done = &v
		case "false", "0", "no":
			v := false
			done = &v
		default:
			shared.WriteError(w, http.StatusBadRequest, "invalid done filter (use true/false)")
			return
		}
	}

	items, err := h.service.List(r.Context(), mediaType, done)
	if err != nil {
		log.Error().Err(err).Msg("list library items failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list library items")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponses(items))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	item, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("get library item failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get library item")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Update(r.Context(), id, req.toInput())
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("update library item failed")
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	shared.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("delete library item failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete library item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
