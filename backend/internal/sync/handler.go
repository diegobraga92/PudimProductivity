package sync

import (
	"context"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// ServeHTTP upgrades the request to a WebSocket connection and streams events.
//
// Query parameters:
//   - last_seq: the client's last seen sequence number. Omitted or 0 on a fresh
//     connection (all buffered events are replayed). If the requested seq is too
//     old for the in-memory replay buffer, the server sends a "stale" event and
//     the client must refetch via REST.
//
// Identity: the connection inherits the authenticated user from the shared
// AuthMiddleware (X-User-ID / X-User-Role dev headers). Presence is published
// on connect/disconnect and events are scoped to the user's task-list
// membership when a MembershipResolver is configured (Phase 8).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lastSeq := int64(0)
	if raw := r.URL.Query().Get("last_seq"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			lastSeq = v
		}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Local dev serves plain HTTP; production sits behind a TLS-terminating
		// proxy. Origin checks are skipped for now (see threat model I2/T1).
		InsecureSkipVerify: true,
	})
	if err != nil {
		_, hijackable := w.(http.Hijacker)
		log.Warn().Err(err).Bool("hijackable", hijackable).Msg("websocket upgrade rejected")
		return
	}
	// Clients send no data; bound read size to control frames only.
	conn.SetReadLimit(1024)

	userID := shared.GetUserID(r.Context())
	role := shared.GetUserRole(r.Context())

	// Resolve task-list membership for event scoping + presence. When the
	// resolver is unavailable the connection degrades to broadcast (legacy).
	listIDs := make(map[string]struct{})
	if h.resolver != nil {
		ids, err := h.resolver.ListIDsForUser(r.Context(), userID, role)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("failed to resolve membership, broadcasting")
		} else {
			for _, id := range ids {
				listIDs[id] = struct{}{}
			}
		}
	}

	remote := r.RemoteAddr
	c := newClient(h, conn, remote, lastSeq, userID, role, listIDs)
	replay, stale, added := h.register(c)
	if !added {
		_ = conn.Close(websocket.StatusPolicyViolation, "server busy")
		return
	}
	defer h.removeClient(c)
	defer c.close()

	log.Info().Str("user_id", userID).Str("remote", remote).Int64("last_seq", lastSeq).Int("lists", len(listIDs)).Msg("ws client connected")

	// Presence: announce the connect, then announce the disconnect when run
	// returns. Both go through the bus so they are replayed to late joiners.
	if userID != "" && userID != "anonymous" {
		h.publishPresence(context.Background(), eventbus.EventPresenceOnline, map[string]any{
			"user_id":  userID,
			"list_ids": c.listIDSlice(),
		})
		defer h.publishPresence(context.Background(), eventbus.EventPresenceOffline, map[string]any{
			"user_id": userID,
		})
	}

	c.run(r.Context(), replay, stale)
	log.Info().Str("user_id", userID).Str("remote", remote).Msg("ws client disconnected")
}
