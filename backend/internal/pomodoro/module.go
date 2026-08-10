package pomodoro

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

func RegisterPomodoroRoutes(r chi.Router, noise NoiseProvider, auditLogger audit.Logger, bus eventbus.Bus) {
	service := NewPomodoroService(noise, auditLogger, bus)
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
