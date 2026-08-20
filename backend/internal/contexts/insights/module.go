package insights

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// RegisterInsightsRoutes wires the Phase 9a insights module. It requires a
// repository; when repo is nil (degraded startup) no routes are registered.
func RegisterInsightsRoutes(r chi.Router, repo InsightRepository, bus eventbus.Bus, flags *featureflag.Service) {
	if repo == nil {
		return
	}
	service := NewInsightService(repo, bus, flags, nil)
	if err := service.Start(context.Background()); err != nil {
		log.Warn().Err(err).Msg("insights failed to subscribe to event bus — focus history not recorded")
	}
	handler := NewHandler(service)

	r.Get("/api/v1/insights/weekly", handler.WeeklyReport)
	log.Info().Msg("insights module routes registered")
}
