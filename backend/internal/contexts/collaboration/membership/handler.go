package membership

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/sync"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// PresenceHandler serves the REST presence snapshot endpoint.
type PresenceHandler struct {
	hub *sync.Hub
}

// NewPresenceHandler builds a handler backed by the sync hub's presence state.
func NewPresenceHandler(hub *sync.Hub) *PresenceHandler {
	return &PresenceHandler{hub: hub}
}

type presenceResponse struct {
	ListID string   `json:"list_id"`
	Online []string `json:"online"`
}

// GET /api/v1/presence/{listId} — the user IDs currently connected and able to
// access the list. Used for the initial presence snapshot; live updates arrive
// via the presence.online/presence.offline WebSocket events.
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

// RegisterCollabRoutes mounts the Phase 8 collaboration endpoints.
//
// GET /api/v1/presence/{listId} — online members snapshot.
func RegisterCollabRoutes(r chi.Router, hub *sync.Hub) {
	handler := NewPresenceHandler(hub)
	r.Get("/api/v1/presence/{listId}", handler.ListPresence)
	log.Info().Msg("collab module routes registered")
}
