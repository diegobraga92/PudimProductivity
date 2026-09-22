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
// root). Every other GET path under /api/v1/sounds/ serves an audio file, while
// PUT/DELETE address a single sound by id and POST adds a new one.
func mountSounds(r chi.Router, h *Handler) {
	r.Get("/api/v1/sounds", h.ListCatalog)
	r.Post("/api/v1/sounds", h.Create)
	r.Route("/api/v1/sounds", func(r chi.Router) {
		r.Get("/", h.ListCatalog)
		r.Post("/", h.Create)
		r.Get("/*", h.GetFile)
		r.Put("/{soundId}", h.Update)
		r.Delete("/{soundId}", h.Delete)
	})
}
