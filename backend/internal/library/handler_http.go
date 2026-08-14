package library

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// maxImportItems caps a single bulk-import request to keep the payload sane.
const maxImportItems = 5000

type Handler struct {
	service Service
	lookup  ScoreLookupClient
	flags   *featureflag.Service
}

func NewHandler(service Service, lookup ScoreLookupClient, flags *featureflag.Service) *Handler {
	if lookup == nil {
		lookup = NoopScoreLookup{}
	}
	return &Handler{service: service, lookup: lookup, flags: flags}
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

// SearchScores looks up ratings for a title from the configured provider.
// Read-only and anonymous like List. Returns 503 when the feature is disabled
// or no provider is configured (graceful degradation, per ADR 007).
func (h *Handler) SearchScores(w http.ResponseWriter, r *http.Request) {
	if h.flags != nil {
		if enabled, err := h.flags.IsEnabled(r.Context(), scoreLookupFeatureFlag); err != nil {
			log.Warn().Err(err).Str("flag", scoreLookupFeatureFlag).Msg("failed to check score lookup flag")
		} else if !enabled {
			shared.WriteError(w, http.StatusServiceUnavailable, "score lookup is disabled")
			return
		}
	}
	if _, ok := h.lookup.(NoopScoreLookup); ok {
		shared.WriteError(w, http.StatusServiceUnavailable, "score lookup is not configured")
		return
	}

	mediaType := MediaType(r.URL.Query().Get("type"))
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		shared.WriteError(w, http.StatusBadRequest, "query parameter is required")
		return
	}
	if mediaType == "" {
		shared.WriteError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if !mediaType.Valid() {
		shared.WriteError(w, http.StatusBadRequest, "invalid type")
		return
	}

	var year *int
	if raw := r.URL.Query().Get("year"); raw != "" {
		if y, err := strconv.Atoi(raw); err == nil && y >= 1800 && y <= 2100 {
			year = &y
		} else {
			shared.WriteError(w, http.StatusBadRequest, "invalid year (use a 4-digit year between 1800 and 2100)")
			return
		}
	}

	cands, err := h.lookup.Search(r.Context(), ScoreQuery{
		Name:        query,
		MediaType:   mediaType,
		ReleaseYear: year,
	})
	if err != nil {
		log.Error().Err(err).Str("type", string(mediaType)).Str("query", query).Msg("score lookup failed")
		shared.WriteError(w, http.StatusBadGateway, "score lookup failed")
		return
	}
	shared.WriteJSON(w, http.StatusOK, toScoreResponses(cands))
}
