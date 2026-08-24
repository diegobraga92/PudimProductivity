package sync

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterSyncRoutes mounts the sync endpoint.
func RegisterSyncRoutes(r chi.Router, hub *Hub) {
	handler := NewHandler(hub)
	r.Get("/api/v1/ws", handler.ServeHTTP)
	log.Info().Msg("sync module routes registered")
}
