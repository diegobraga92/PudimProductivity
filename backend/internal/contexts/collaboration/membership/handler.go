package membership

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/sync"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// TODO: Add proper users, login and auth to make use of membership

// PresenceHandler serves the sync presence endpoints.
type PresenceHandler struct {
	hub *sync.Hub
}

// NewPresenceHandler builds a presence handler.
func NewPresenceHandler(hub *sync.Hub) *PresenceHandler {
	return &PresenceHandler{hub: hub}
}

type presenceResponse struct {
	ListID string   `json:"list_id"`
	Online []string `json:"online"`
}

// ListPresence for the GET endpoint /presence/{listId}
// The user IDs currently connected and able to access the list
func (h *PresenceHandler) ListPresence(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "listId")
	if listID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	online := h.hub.OnlineUsersForList(listID)
	if online == nil {
		online = make([]string, 0)
	}
	httpx.WriteJSON(w, http.StatusOK, presenceResponse{ListID: listID, Online: online})
}

// RegisterCollabRoutes mounts collaboration endpoints.
func RegisterCollabRoutes(r chi.Router, hub *sync.Hub) {
	handler := NewPresenceHandler(hub)
	r.Get("/api/v1/presence/{listId}", handler.ListPresence)
	log.Info().Msg("collab module routes registered")
}
