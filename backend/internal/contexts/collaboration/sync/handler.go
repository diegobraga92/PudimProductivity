package sync

import (
	"context"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// Handler is the HTTP transport for the real-time sync endpoint.
type Handler struct {
	hub *Hub
}

// NewHandler builds a sync handler bound to the given hub.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP upgrades the request to a WebSocket connection and streams events.
// last_seq is the client's last seen sequence number. Omitted or 0 on a fresh connection.
// If seq is too old, the server sends a "stale" event and the client must refetch via REST.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lastSeq := int64(0)
	if raw := r.URL.Query().Get("last_seq"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			lastSeq = v
		}
	}

	// TODO: Local dev serves plain HTTP. Origin checks are skipped for now.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		_, hijackable := w.(http.Hijacker)
		log.Warn().Err(err).Bool("hijackable", hijackable).Msg("websocket upgrade rejected")
		return
	}

	// Bound read size to control frames only.
	conn.SetReadLimit(1024)

	userID := httpx.GetUserID(r.Context())
	role := httpx.GetUserRole(r.Context())

	// Resolve task-list membership for event scoping + presence. Degrades to broadcast.
	listIDs := make(map[string]struct{})
	if h.hub.resolver != nil {
		ids, err := h.hub.resolver.ListIDsForUser(r.Context(), userID, role)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("failed to resolve membership, broadcasting")
		} else {
			for _, id := range ids {
				listIDs[id] = struct{}{}
			}
		}
	}

	remote := r.RemoteAddr
	c := newClient(h.hub, conn, remote, lastSeq, userID, role, listIDs)
	replay, stale, added := h.hub.register(c)
	if !added {
		_ = conn.Close(websocket.StatusPolicyViolation, "server busy")
		return
	}
	defer h.hub.removeClient(c)
	defer c.close()

	log.Info().Str("user_id", userID).Str("remote", remote).Int64("last_seq", lastSeq).Int("lists", len(listIDs)).Msg("ws client connected")

	// Presence: announce the connect, then announce the disconnect when run
	// returns. Both go through the bus so they are replayed to late joiners.
	if userID != "" && userID != "anonymous" {
		h.hub.publishPresence(context.Background(), eventbus.EventPresenceOnline, map[string]any{
			"user_id":  userID,
			"list_ids": c.listIDSlice(),
		})
		defer h.hub.publishPresence(context.Background(), eventbus.EventPresenceOffline, map[string]any{
			"user_id": userID,
		})
	}

	c.run(r.Context(), replay, stale)
	log.Info().Str("user_id", userID).Str("remote", remote).Msg("ws client disconnected")
}
