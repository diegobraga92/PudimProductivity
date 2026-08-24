package backup

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// RegisterBackupRoutes wires the Backup module onto the router.
// Endpoints require admin due to the nature of the operations.
func RegisterBackupRoutes(r chi.Router, repo Repository, appVersion string) {
	handler := NewHandler(repo, appVersion)

	r.Route("/api/v1/backup", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(httpx.RequireRole("admin", "user"))
			r.Get("/export", handler.Export)
			r.Post("/import", handler.Import)
		})
	})

	log.Info().Msg("backup module routes registered")
}
