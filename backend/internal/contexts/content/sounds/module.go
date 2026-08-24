package sounds

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterSoundsRoutes mounts the Soundscape sound library endpoints:
//   - GET /api/v1/sounds      — catalog of available sounds (id → file)
//   - GET /api/v1/sounds/{file} — audio file bytes (Range-capable)
//
// The exact and trailing-slash catalog paths are both registered because the
// web client calls /api/v1/sounds (no trailing slash).
func RegisterSoundsRoutes(r chi.Router, dir string, catalog []Sound) {
	h := NewHandler(dir, catalog)
	r.Get("/api/v1/sounds", h.ListCatalog)
	r.Route("/api/v1/sounds", func(r chi.Router) {
		r.Get("/", h.ListCatalog)
		r.Get("/*", h.GetFile)
	})
	log.Info().Str("dir", dir).Int("sounds", len(catalog)).Msg("soundscape sound routes registered")
}
