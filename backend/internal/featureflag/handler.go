package featureflag

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service *Service
	audit   audit.Logger
}

func NewHandler(service *Service, auditLogger audit.Logger) *Handler {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &Handler{service: service, audit: auditLogger}
}

type featureFlagResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type toggleFlagRequest struct {
	Enabled bool `json:"enabled"`
}

func toFlagResponse(f FeatureFlag) featureFlagResponse {
	return featureFlagResponse{
		ID:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		Enabled:     f.Enabled,
	}
}

// GET /api/v1/features
func (h *Handler) ListEnabled(w http.ResponseWriter, r *http.Request) {
	flags, err := h.service.ListEnabled(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list enabled feature flags")
		shared.WriteError(w, http.StatusInternalServerError, "failed to list feature flags")
		return
	}

	responses := make([]featureFlagResponse, len(flags))
	for i, f := range flags {
		responses[i] = toFlagResponse(f)
	}

	shared.WriteJSON(w, http.StatusOK, responses)
}

// GET /api/v1/features/{name}
func (h *Handler) GetByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		shared.WriteError(w, http.StatusBadRequest, "flag name is required")
		return
	}

	enabled, err := h.service.IsEnabled(r.Context(), name)
	if err != nil {
		log.Error().Err(err).Str("flag", name).Msg("failed to get feature flag")
		shared.WriteError(w, http.StatusInternalServerError, "failed to get feature flag")
		return
	}

	shared.WriteJSON(w, http.StatusOK, featureFlagResponse{
		Name:    name,
		Enabled: enabled,
	})
}

// PUT /api/v1/features/{name}/toggle
func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		shared.WriteError(w, http.StatusBadRequest, "flag name is required")
		return
	}

	var req toggleFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	flag, err := h.service.GetByName(r.Context(), name)
	if err != nil {
		log.Error().Err(err).Str("flag", name).Msg("failed to look up feature flag")
		shared.WriteError(w, http.StatusInternalServerError, "failed to look up feature flag")
		return
	}
	if flag == nil {
		shared.WriteError(w, http.StatusNotFound, "feature flag not found")
		return
	}

	if err := h.service.SetEnabled(r.Context(), flag.ID, req.Enabled); err != nil {
		log.Error().Err(err).Str("flag", name).Msg("failed to toggle feature flag")
		shared.WriteError(w, http.StatusInternalServerError, "failed to toggle feature flag")
		return
	}

	h.audit.Log(r.Context(), audit.ActionFeatureToggled, audit.ResourceFeatures, flag.ID, map[string]any{
		"enabled": !req.Enabled,
	}, map[string]any{
		"enabled": req.Enabled,
	})

	shared.WriteJSON(w, http.StatusOK, featureFlagResponse{
		ID:      flag.ID,
		Name:    flag.Name,
		Enabled: req.Enabled,
	})
}
