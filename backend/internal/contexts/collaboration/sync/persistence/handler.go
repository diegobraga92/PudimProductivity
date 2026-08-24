package persistence

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Bundle is the full incremental payload of the sync endpoint
// (api/openapi/sync-v1.yaml). It is composed of the canonical API response
// types so the offline wire shape always matches the task/tasklist endpoints.
type Bundle struct {
	Timestamp            string                        `json:"timestamp"`
	Tasks                []task.TaskResponse           `json:"tasks"`
	DeletedTaskIDs       []string                      `json:"deleted_task_ids"`
	Completions          []task.TaskCompletionResponse `json:"completions"`
	DeletedCompletionIDs []string                      `json:"deleted_completion_ids"`
	TaskLists            []tasklist.TaskListResponse   `json:"task_lists"`
	DeletedTaskListIDs   []string                      `json:"deleted_task_list_ids"`
	Shares               []tasklist.MemberResponse     `json:"shares"`
	DeletedShareKeys     []string                      `json:"deleted_share_keys"`
}

// toBundle maps the domain ChangeSet to the wire Bundle using the canonical
// response mappers (task.ToTaskResponse, tasklist.ToTaskListResponse, ...).
// All slices are normalized to non-nil so the JSON payload emits [] instead of
// null.
func toBundle(cs *ChangeSet) *Bundle {
	b := &Bundle{
		Timestamp:            cs.Timestamp.UTC().Format(time.RFC3339),
		Tasks:                make([]task.TaskResponse, 0, len(cs.Tasks)),
		DeletedTaskIDs:       make([]string, 0, len(cs.DeletedTaskIDs)),
		Completions:          make([]task.TaskCompletionResponse, 0, len(cs.Completions)),
		DeletedCompletionIDs: make([]string, 0, len(cs.DeletedCompletionIDs)),
		TaskLists:            make([]tasklist.TaskListResponse, 0, len(cs.TaskLists)),
		DeletedTaskListIDs:   make([]string, 0, len(cs.DeletedTaskListIDs)),
		Shares:               make([]tasklist.MemberResponse, 0, len(cs.Shares)),
		DeletedShareKeys:     make([]string, 0, len(cs.DeletedShareKeys)),
	}
	for _, t := range cs.Tasks {
		b.Tasks = append(b.Tasks, task.ToTaskResponse(t))
	}
	for _, c := range cs.Completions {
		b.Completions = append(b.Completions, task.ToTaskCompletionResponse(c))
	}
	for _, l := range cs.TaskLists {
		b.TaskLists = append(b.TaskLists, tasklist.ToTaskListResponse(l))
	}
	for _, s := range cs.Shares {
		b.Shares = append(b.Shares, tasklist.ToMemberResponse(s))
	}
	return b
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
			httpx.WriteError(w, http.StatusBadRequest, "since must be an RFC3339 timestamp")
			return
		}
		since = parsed.UTC()
	}

	bundle, err := h.repo.Bundle(r.Context(), since)
	if err != nil {
		log.Error().Err(err).Msg("failed to build sync bundle")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to sync")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toBundle(bundle))
}

// RegisterSyncStoreRoutes mounts the Phase 9c offline-sync endpoint.
func RegisterSyncStoreRoutes(r chi.Router, repo Repository) {
	if repo == nil {
		return
	}
	handler := NewHandler(repo)
	r.Get("/api/v1/sync", handler.Sync)
	log.Info().Msg("syncstore module routes registered")
}
