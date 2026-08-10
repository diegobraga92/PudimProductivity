package syncstore

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// GET /api/v1/sync?since=2026-08-10T10:00:00Z
//
// since is an RFC3339 timestamp (client's last sync time). Defaults to the
// epoch (full snapshot). The response's `timestamp` should be persisted by the
// client and sent back as `since` next time.
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	since := time.Time{}
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "since must be an RFC3339 timestamp")
			return
		}
		since = parsed.UTC()
	}

	bundle, err := h.repo.Bundle(r.Context(), since)
	if err != nil {
		log.Error().Err(err).Msg("failed to build sync bundle")
		shared.WriteError(w, http.StatusInternalServerError, "failed to sync")
		return
	}

	shared.WriteJSON(w, http.StatusOK, bundle)
}

// RegisterSyncStoreRoutes mounts the Phase 9c offline-sync endpoint.
func RegisterSyncStoreRoutes(r chi.Router, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	repo := NewPostgresRepository(pool)
	handler := NewHandler(repo)
	r.Get("/api/v1/sync", handler.Sync)
	log.Info().Msg("syncstore module routes registered")
}
