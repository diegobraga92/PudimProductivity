package recipe

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recipe, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		log.Error().Err(err).Msg("create recipe failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toResponse(recipe))
}

func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{
		Search:     r.URL.Query().Get("search"),
		Difficulty: r.URL.Query().Get("difficulty"),
	}
	if tags := r.URL.Query().Get("tags"); tags != "" {
		for _, t := range strings.Split(tags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				filter.Tags = append(filter.Tags, t)
			}
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	// Cursor format: <RFC3339 createdAt>,<id> (opaque to clients).
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		parts := strings.SplitN(cursor, ",", 2)
		if len(parts) == 2 {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				filter.Cursor = &t
				filter.CursorID = parts[1]
			}
		}
	}

	recipes, err := h.service.List(r.Context(), filter)
	if err != nil {
		log.Error().Err(err).Msg("list recipes failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list recipes")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponses(recipes))
}

func (h *Handler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "recipeId")
	recipe, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "recipe not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("recipe_id", id).Msg("get recipe failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get recipe")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(recipe))
}

func (h *Handler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "recipeId")
	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recipe, err := h.service.Update(r.Context(), id, req.toInput())
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "recipe not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("recipe_id", id).Msg("update recipe failed")
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(recipe))
}

func (h *Handler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "recipeId")
	err := h.service.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "recipe not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("recipe_id", id).Msg("delete recipe failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete recipe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GenerateUploadURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "recipeId")
	var req UploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ContentType == "" {
		httpx.WriteError(w, http.StatusBadRequest, "content_type is required")
		return
	}

	upload, err := h.service.GenerateUploadURL(r.Context(), id, req.ContentType, req.Filename)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "recipe not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("recipe_id", id).Msg("generate upload URL failed")
		httpx.WriteError(w, http.StatusServiceUnavailable, "media uploads not available")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, upload)
}
