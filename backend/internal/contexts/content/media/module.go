package media

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterMediaRoutes mounts the local filesystem media endpoints.
func RegisterMediaRoutes(r chi.Router, dir string) {
	h := NewMediaHandler(dir)
	r.Route("/api/v1/media", func(r chi.Router) {
		r.Put("/*", h.Put)
		r.Get("/*", h.Get)
	})
	log.Info().Msg("local media routes registered")
}
