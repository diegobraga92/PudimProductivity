package tasklist

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

type Handler struct {
	service *TaskListService
}

func NewHandler(service *TaskListService) *Handler {
	return &Handler{service: service}
}

// TaskListResponse is the canonical wire shape for a task list.
type TaskListResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type createTaskListRequest struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type updateTaskListRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type shareTaskListRequest struct {
	SharedWith string `json:"shared_with"`
	Role       string `json:"role"`
}

// MemberResponse is the canonical wire shape for a task-list share/membership.
type MemberResponse struct {
	ListID     string `json:"list_id"`
	SharedWith string `json:"shared_with"`
	Role       string `json:"role"`
	CreatedAt  string `json:"created_at"`
}

// ToTaskListResponse maps a list to its canonical wire shape.
func ToTaskListResponse(l *TaskList) TaskListResponse {
	return TaskListResponse{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		OwnerID:     l.OwnerID,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ToMemberResponse maps a share to its canonical wire shape.
func ToMemberResponse(s *Share) MemberResponse {
	return MemberResponse{
		ListID:     s.ListID,
		SharedWith: s.SharedWith,
		Role:       string(s.Role),
		CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// isAdmin reports whether the request was made with the admin role.
func isAdmin(r *http.Request) bool {
	return httpx.GetUserRole(r.Context()) == "admin"
}

// requester returns the authenticated user ID from the dev identity headers.
func requester(r *http.Request) string {
	return httpx.GetUserID(r.Context())
}

// GET /api/v1/task-lists — returns the lists the requesting user can access
// (owned + shared with them). Admins see all lists.
func (h *Handler) ListTaskLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.service.ListTaskListsForUser(r.Context(), requester(r), isAdmin(r))
	if err != nil {
		log.Error().Err(err).Msg("failed to list task lists")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list task lists")
		return
	}

	responses := make([]TaskListResponse, len(lists))
	for i, l := range lists {
		responses[i] = ToTaskListResponse(l)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}

// POST /api/v1/task-lists
func (h *Handler) CreateTaskList(w http.ResponseWriter, r *http.Request) {
	var req createTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := h.service.CreateTaskList(r.Context(), req.ID, req.Name, requester(r))
	if err != nil {
		log.Error().Err(err).Msg("failed to create task list")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to create task list")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, ToTaskListResponse(list))
}

// GET /api/v1/task-lists/{listId}
func (h *Handler) GetTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	list, err := h.service.GetTaskListForUser(r.Context(), id, requester(r), isAdmin(r))
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		if errors.Is(err, ErrTaskListAccessDenied) {
			httpx.WriteError(w, http.StatusForbidden, "access denied to task list")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to get task list")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get task list")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ToTaskListResponse(list))
}

// POST /api/v1/task-lists/{listId}/share — invites a user to the list.
func (h *Handler) ShareTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	var req shareTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SharedWith == "" {
		httpx.WriteError(w, http.StatusBadRequest, "shared_with is required")
		return
	}

	role := Role(req.Role)
	if role != RoleEditor && role != RoleViewer {
		httpx.WriteError(w, http.StatusBadRequest, "role must be 'editor' or 'viewer'")
		return
	}

	if err := h.service.ShareList(r.Context(), id, requester(r), req.SharedWith, role, isAdmin(r)); err != nil {
		switch {
		case errors.Is(err, ErrTaskListNotFound):
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
		case errors.Is(err, ErrTaskListAccessDenied):
			httpx.WriteError(w, http.StatusForbidden, "only the owner can share this list")
		case errors.Is(err, ErrShareExists):
			httpx.WriteError(w, http.StatusConflict, "user is already a member of this list")
		default:
			log.Error().Err(err).Str("list_id", id).Msg("failed to share task list")
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/task-lists/{listId}/share/{userId} — revokes access.
func (h *Handler) UnshareTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	userID := chi.URLParam(r, "userId")
	if id == "" || userID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID and user ID are required")
		return
	}

	if err := h.service.UnshareList(r.Context(), id, requester(r), userID, isAdmin(r)); err != nil {
		switch {
		case errors.Is(err, ErrTaskListNotFound):
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
		case errors.Is(err, ErrTaskListAccessDenied):
			httpx.WriteError(w, http.StatusForbidden, "only the owner can unshare this list")
		case errors.Is(err, ErrShareNotFound):
			httpx.WriteError(w, http.StatusNotFound, "share not found")
		default:
			log.Error().Err(err).Str("list_id", id).Msg("failed to unshare task list")
			httpx.WriteError(w, http.StatusInternalServerError, "failed to unshare task list")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/task-lists/{listId}/members — lists the shared members of a list.
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	shares, err := h.service.ListMembers(r.Context(), id, requester(r), isAdmin(r))
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		if errors.Is(err, ErrTaskListAccessDenied) {
			httpx.WriteError(w, http.StatusForbidden, "access denied to task list")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to list members")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	responses := make([]MemberResponse, len(shares))
	for i, s := range shares {
		responses[i] = ToMemberResponse(s)
	}
	httpx.WriteJSON(w, http.StatusOK, responses)
}

// PUT /api/v1/task-lists/{listId}
func (h *Handler) UpdateTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	var req updateTaskListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	list, err := h.service.UpdateTaskList(r.Context(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to update task list")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to update task list")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ToTaskListResponse(list))
}

// DELETE /api/v1/task-lists/{listId}
func (h *Handler) DeleteTaskList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "listId")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
		return
	}

	// Only the owner (or an admin) may delete a list (Phase 8).
	if err := h.service.CheckAccess(r.Context(), id, requester(r), RoleOwner, isAdmin(r)); err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		if errors.Is(err, ErrTaskListAccessDenied) {
			httpx.WriteError(w, http.StatusForbidden, "only the owner can delete this list")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to check access to task list")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete task list")
		return
	}

	if err := h.service.DeleteTaskList(r.Context(), id); err != nil {
		if errors.Is(err, ErrTaskListNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "task list not found")
			return
		}
		log.Error().Err(err).Str("list_id", id).Msg("failed to delete task list")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to delete task list")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/task-lists/{listId}/tasks
// Returns a handler that lists tasks for a list. Uses task service to fetch tasks by list ID
func (h *Handler) ListTasksByListID(taskService task.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "listId")
		if id == "" {
			httpx.WriteError(w, http.StatusBadRequest, "list ID is required")
			return
		}

		// Members (owner, editor, viewer) can read tasks (Phase 8).
		if err := h.service.CheckAccess(r.Context(), id, requester(r), RoleViewer, isAdmin(r)); err != nil {
			if errors.Is(err, ErrTaskListNotFound) {
				httpx.WriteError(w, http.StatusNotFound, "task list not found")
				return
			}
			if errors.Is(err, ErrTaskListAccessDenied) {
				httpx.WriteError(w, http.StatusForbidden, "access denied to task list")
				return
			}
			log.Error().Err(err).Str("list_id", id).Msg("failed to check access to task list")
			httpx.WriteError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		typeFilter := r.URL.Query().Get("type")

		tasks, err := taskService.ListTasksByListID(r.Context(), id, typeFilter)
		if err != nil {
			log.Error().Err(err).Str("list_id", id).Msg("failed to list tasks by list")
			httpx.WriteError(w, http.StatusInternalServerError, "failed to list tasks")
			return
		}

		responses := make([]task.TaskResponse, len(tasks))
		for i, t := range tasks {
			responses[i] = task.ToTaskResponse(t)
		}

		httpx.WriteJSON(w, http.StatusOK, responses)
	}
}
