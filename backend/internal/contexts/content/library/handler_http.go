package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

const maxImportItems = 5000
const maxScoreBatchItems = 100
const batchWorkers = 5

type Handler struct {
	service Service
	lookup  ScoreLookupProvider
	flags   *featureflag.Service
}

func NewHandler(service Service, lookup ScoreLookupProvider, flags *featureflag.Service) *Handler {
	if lookup == nil {
		lookup = NoopScoreLookup{}
	}
	return &Handler{service: service, lookup: lookup, flags: flags}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		log.Error().Err(err).Msg("create library item failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toResponse(item))
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "no items to import")
		return
	}
	if len(req.Items) > maxImportItems {
		httpx.WriteError(w, http.StatusBadRequest, "too many items in a single import")
		return
	}

	inputs := make([]CreateInput, 0, len(req.Items))
	for _, it := range req.Items {
		inputs = append(inputs, it.toInput())
	}
	result, err := h.service.Import(r.Context(), inputs)
	if err != nil {
		log.Error().Err(err).Msg("import library items failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to import library items")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
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
			httpx.WriteError(w, http.StatusBadRequest, "invalid done filter (use true/false)")
			return
		}
	}

	items, err := h.service.List(r.Context(), ListFilter{
		MediaType: mediaType,
		Done:      done,
		Subtype:   strings.TrimSpace(r.URL.Query().Get("subtype")),
	})
	if err != nil {
		log.Error().Err(err).Msg("list library items failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list library items")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponses(items))
}

// Subtypes returns the distinct non-empty genre/console values used by the
// library's subtype filter dropdown.
func (h *Handler) Subtypes(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	if mediaType != "" && !MediaType(mediaType).Valid() {
		httpx.WriteError(w, http.StatusBadRequest, "invalid type")
		return
	}
	subtypes, err := h.service.DistinctSubtypes(r.Context(), mediaType)
	if err != nil {
		log.Error().Err(err).Msg("list library subtypes failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list library subtypes")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, subtypes)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	item, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("get library item failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get library item")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Update(r.Context(), id, req.toInput())
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("update library item failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "itemId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "library item not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("item_id", id).Msg("delete library item failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete library item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// scoreLookupReady writes a 503 when the score-lookup feature is disabled or
// no provider is configured, and reports whether the caller may proceed.
func (h *Handler) scoreLookupReady(w http.ResponseWriter, r *http.Request) bool {
	if h.flags != nil {
		if enabled, err := h.flags.IsEnabled(r.Context(), ScoreLookupFeatureFlag); err != nil {
			log.Warn().Err(err).Str("flag", ScoreLookupFeatureFlag).Msg("failed to check score lookup flag")
		} else if !enabled {
			httpx.WriteError(w, http.StatusServiceUnavailable, "score lookup is disabled")
			return false
		}
	}
	if h.lookup == nil || !h.lookup.Configured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "score lookup is not configured")
		return false
	}
	return true
}

// SearchScores looks up ratings for a title from the configured provider.
func (h *Handler) SearchScores(w http.ResponseWriter, r *http.Request) {
	if !h.scoreLookupReady(w, r) {
		return
	}

	mediaType := MediaType(r.URL.Query().Get("type"))
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		httpx.WriteError(w, http.StatusBadRequest, "query parameter is required")
		return
	}
	if mediaType == "" {
		httpx.WriteError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if !mediaType.Valid() {
		httpx.WriteError(w, http.StatusBadRequest, "invalid type")
		return
	}

	var year *int
	if raw := r.URL.Query().Get("year"); raw != "" {
		if y, err := strconv.Atoi(raw); err == nil && y >= 1800 && y <= 2100 {
			year = &y
		} else {
			httpx.WriteError(w, http.StatusBadRequest, "invalid year (use a 4-digit year between 1800 and 2100)")
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
		httpx.WriteError(w, http.StatusBadGateway, "score lookup failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toScoreResponses(cands))
}

// SearchScoresBatch performs score lookups for many titles at once (used by the
// CSV import auto-scoring flow).
func (h *Handler) SearchScoresBatch(w http.ResponseWriter, r *http.Request) {
	if !h.scoreLookupReady(w, r) {
		return
	}

	var req ScoreBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "no items to look up")
		return
	}
	if len(req.Items) > maxScoreBatchItems {
		httpx.WriteError(w, http.StatusBadRequest, "too many items in a single batch (max 100)")
		return
	}

	results := make([]ScoreBatchResult, len(req.Items))
	sem := make(chan struct{}, batchWorkers)
	var wg sync.WaitGroup
	for i, item := range req.Items {
		wg.Add(1)
		go func(i int, item ScoreBatchItemRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = h.lookupBatchItem(r.Context(), i, item)
		}(i, item)
	}
	wg.Wait()

	httpx.WriteJSON(w, http.StatusOK, ScoreBatchResponse{Results: results})
}

// lookupBatchItem resolves a single batch row against the configured provider.
func (h *Handler) lookupBatchItem(ctx context.Context, idx int, item ScoreBatchItemRequest) ScoreBatchResult {
	mt := MediaType(item.MediaType)
	if !mt.Valid() {
		return ScoreBatchResult{Index: idx, Error: fmt.Sprintf("invalid type %q", item.MediaType)}
	}
	var year *int
	if item.Year != nil && *item.Year >= 1800 && *item.Year <= 2100 {
		year = item.Year
	}
	cands, err := h.lookup.Search(ctx, ScoreQuery{
		Name:        strings.TrimSpace(item.Name),
		MediaType:   mt,
		ReleaseYear: year,
	})
	if err != nil {
		log.Warn().Err(err).Str("type", string(mt)).Str("query", item.Name).Msg("batch score lookup failed for item")
		return ScoreBatchResult{Index: idx, Error: err.Error()}
	}
	return ScoreBatchResult{Index: idx, Candidates: toScoreResponses(cands)}
}
