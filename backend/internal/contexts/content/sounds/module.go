package sounds

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterSoundsRoutes mounts the Soundscape sound library endpoints.
func RegisterSoundsRoutes(r chi.Router, dir string, catalog []Sound) {
	h := NewHandler(dir, catalog)
	mountSounds(r, h)
	log.Info().Str("dir", dir).Int("sounds", len(catalog)).Msg("soundscape sound routes registered")
}

// mountSounds registers the sound library routes in one place.
//
// The catalog answers both /api/v1/sounds and /api/v1/sounds/ (the subrouter
// root), while every other path under /api/v1/sounds/ serves an audio file.
func mountSounds(r chi.Router, h *Handler) {
	r.Get("/api/v1/sounds", h.ListCatalog)
	r.Route("/api/v1/sounds", func(r chi.Router) {
		r.Get("/", h.ListCatalog)
		r.Get("/*", h.GetFile)
	})
}
