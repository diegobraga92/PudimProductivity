package sync

import (
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
)

// ServeHTTP upgrades the request to a WebSocket connection and streams events.
//
// Query parameters:
//   - last_seq: the client's last seen sequence number. Omitted or 0 on a fresh
//     connection (all buffered events are replayed). If the requested seq is too
//     old for the in-memory replay buffer, the server sends a "stale" event and
//     the client must refetch via REST.
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

	remote := r.RemoteAddr
	c := newClient(h, conn, remote, lastSeq)
	replay, stale, added := h.register(c)
	if !added {
		_ = conn.Close(websocket.StatusPolicyViolation, "server busy")
		return
	}
	defer h.removeClient(c)
	defer c.close()

	log.Info().Str("remote", remote).Int64("last_seq", lastSeq).Msg("ws client connected")
	c.run(r.Context(), replay, stale)
	log.Info().Str("remote", remote).Msg("ws client disconnected")
}
