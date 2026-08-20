package scoring

import (
	"encoding/json"
	"net/http"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// UpdateProviderRequest carries changes for a single provider row.
// APIKey/BaseURL are pointers: nil = keep the stored value, "" = clear it.
type UpdateProviderRequest struct {
	Name    string  `json:"name"`
	APIKey  *string `json:"api_key"`
	BaseURL *string `json:"base_url"`
}

// UpdateConfigRequest is the payload for PUT /api/v1/admin/score-providers.
// LookupEnabled is optional (nil = leave the feature flag unchanged).
type UpdateConfigRequest struct {
	MovieProvider  string                  `json:"movie_provider"`
	SeriesProvider string                  `json:"series_provider"`
	GameProvider   string                  `json:"game_provider"`
	BookProvider   string                  `json:"book_provider"`
	LookupEnabled  *bool                   `json:"lookup_enabled"`
	Providers      []UpdateProviderRequest `json:"providers"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Get returns the effective score-provider configuration (API keys masked).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.Config(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to load score provider settings")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cfg)
}

// Update validates and applies a new configuration.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg, err := h.service.Update(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cfg)
}
