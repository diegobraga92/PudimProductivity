package backup

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// RegisterBackupRoutes wires the Backup module onto the router.
//
// Both endpoints require an authenticated user, matching every other mutating
// route in the app (the dev identity headers X-User-ID / X-User-Role). Export
// is idempotent and Import is the destructive counterpart, so the UI prompts
// for confirmation before posting a file.
func RegisterBackupRoutes(r chi.Router, pool *pgxpool.Pool, appVersion string) {
	service := NewService(pool)
	handler := NewHandler(service, appVersion)

	r.Route("/api/v1/backup", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(shared.RequireRole("admin", "user"))
			r.Get("/export", handler.Export)
			r.Post("/import", handler.Import)
		})
	})

	log.Info().Msg("backup module routes registered")
}
