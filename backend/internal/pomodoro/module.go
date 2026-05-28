package pomodoro

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func RegisterPomodoroRoutes(r chi.Router, noise NoiseProvider) {
	service := NewPomodoroService(noise)
	handler := NewHandler(service)

	r.Route("/api/v1/pomodoro", func(r chi.Router) {
		r.Post("/start", handler.StartSession)
		r.Get("/current", handler.GetCurrent)
		r.Post("/pause", handler.Pause)
		r.Post("/resume", handler.Resume)
		r.Post("/stop", handler.Stop)
	})

	log.Info().Msg("pomodoro module routes registered")
}
