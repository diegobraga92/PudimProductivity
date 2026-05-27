package features

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Handler contains HTTP handlers for feature flag operations.
type Handler struct {
	store *CachedFeatureStore
}

// NewHandler creates a new Handler.
func NewHandler(store *CachedFeatureStore) *Handler {
	return &Handler{store: store}
}

// ListFeatures handles GET /api/v1/features
func (h *Handler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	flags, err := h.store.GetAll(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list feature flags")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list features")
		return
	}

	shared.WriteJSON(w, http.StatusOK, flags)
}

// RegisterFeatureRoutes mounts all feature flag HTTP routes on the given router.
func RegisterFeatureRoutes(r chi.Router, store *CachedFeatureStore) {
	handler := NewHandler(store)
	r.Get("/api/v1/features", handler.ListFeatures)

	log.Info().Msg("feature flag routes registered")
}
