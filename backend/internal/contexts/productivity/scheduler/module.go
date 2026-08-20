package scheduler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RegisterSchedulerRoutes wires the auto-scheduler module. tasks must be the
// task service (satisfies TaskReader).
func RegisterSchedulerRoutes(r chi.Router, tasks TaskReader) *SchedulerService {
	service := NewSchedulerService(tasks)
	handler := NewHandler(service)

	r.Route("/api/v1/schedule", func(r chi.Router) {
		r.Get("/", handler.GetSchedule)
	})

	log.Info().Msg("scheduler module routes registered")
	return service
}
