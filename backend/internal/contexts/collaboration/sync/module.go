package sync

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterSyncRoutes mounts the real-time sync endpoint.
//
// GET /api/v1/ws?last_seq=N — WebSocket stream of task events (see
// api/ws/events-v1.json for the message schema).
func RegisterSyncRoutes(r chi.Router, hub *Hub) {
	r.Get("/api/v1/ws", hub.ServeHTTP)
	log.Info().Msg("sync module routes registered")
}
