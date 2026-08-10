package tasklist

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type TaskListService struct {
	repo TaskListRepository
	bus  eventbus.Bus
}

func NewTaskListService(repo TaskListRepository, bus eventbus.Bus) *TaskListService {
	return &TaskListService{repo: repo, bus: bus}
}

// publish emits a domain event. A nil bus is a no-op; failures are logged, not
// propagated (the DB write is the source of truth).
func (s *TaskListService) publish(ctx context.Context, typ eventbus.EventType, payload interface{}) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish task list event")
	}
}

// effectiveRole returns the user's role on a list. Admins are treated as
// owners; otherwise the owner/share role is used.
func (s *TaskListService) effectiveRole(ctx context.Context, listID, userID string, isAdmin bool) (Role, error) {
	if isAdmin {
		return RoleOwner, nil
	}
	role, err := s.repo.GetMemberRole(ctx, listID, userID)
	if err != nil {
		return "", err
	}
	return role, nil
}

func (s *TaskListService) CreateTaskList(ctx context.Context, name, ownerID string) (*TaskList, error) {
	id := shared.NewUUID()

	list, err := NewTaskList(id, name)
	if err != nil {
		return nil, fmt.Errorf("create task list: %w", err)
	}
	list.OwnerID = ownerID

	if err := s.repo.Create(ctx, list); err != nil {
		return nil, fmt.Errorf("persist task list: %w", err)
	}

	log.Info().Str("list_id", list.ID).Str("name", list.Name).Str("owner", ownerID).Msg("task list created")
	return list, nil
}

func (s *TaskListService) GetTaskList(ctx context.Context, id string) (*TaskList, error) {
	list, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetTaskListForUser returns a list only if userID can access it.
func (s *TaskListService) GetTaskListForUser(ctx context.Context, id, userID string, isAdmin bool) (*TaskList, error) {
	if _, err := s.effectiveRole(ctx, id, userID, isAdmin); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *TaskListService) ListTaskLists(ctx context.Context) ([]*TaskList, error) {
	return s.repo.List(ctx)
}

// ListTaskListsForUser returns lists the user owns or is a member of (Phase 8).
func (s *TaskListService) ListTaskListsForUser(ctx context.Context, userID string, isAdmin bool) ([]*TaskList, error) {
	if isAdmin {
		return s.repo.List(ctx)
	}
	return s.repo.ListListsForUser(ctx, userID)
}

// ShareList grants sharedWith editor or viewer access. Only the owner (or an
// admin) may share; you cannot share a list with yourself or with the owner.
func (s *TaskListService) ShareList(ctx context.Context, listID, sharedBy, sharedWith string, role Role, isAdmin bool) error {
	current, err := s.effectiveRole(ctx, listID, sharedBy, isAdmin)
	if err != nil {
		return err
	}
	if !current.AtLeast(RoleOwner) {
		return ErrTaskListAccessDenied
	}
	if sharedWith == sharedBy {
		return fmt.Errorf("cannot share a list with yourself")
	}
	// Reject sharing with the owner.
	ownerRole, err := s.repo.GetMemberRole(ctx, listID, sharedWith)
	if err == nil && ownerRole == RoleOwner {
		return fmt.Errorf("user already owns this list")
	}
	if err != nil && !errors.Is(err, ErrTaskListAccessDenied) {
		return err
	}

	share, err := NewShare(listID, sharedWith, role)
	if err != nil {
		return err
	}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return err
	}

	log.Info().Str("list_id", listID).Str("shared_with", sharedWith).Str("role", string(role)).Msg("task list shared")
	s.publish(ctx, eventbus.EventTaskListShared, map[string]any{
		"list_id":     listID,
		"shared_with": sharedWith,
		"role":        string(role),
		"shared_by":   sharedBy,
	})
	return nil
}

// UnshareList revokes a share. Only the owner (or an admin) may unshare.
func (s *TaskListService) UnshareList(ctx context.Context, listID, sharedBy, sharedWith string, isAdmin bool) error {
	current, err := s.effectiveRole(ctx, listID, sharedBy, isAdmin)
	if err != nil {
		return err
	}
	if !current.AtLeast(RoleOwner) {
		return ErrTaskListAccessDenied
	}
	if err := s.repo.DeleteShare(ctx, listID, sharedWith); err != nil {
		return err
	}

	log.Info().Str("list_id", listID).Str("shared_with", sharedWith).Msg("task list unshared")
	s.publish(ctx, eventbus.EventTaskListUnshared, map[string]any{
		"list_id":     listID,
		"shared_with": sharedWith,
		"removed_by":  sharedBy,
	})
	return nil
}

// ListMembers returns the members of a list. Requires at least viewer access.
func (s *TaskListService) ListMembers(ctx context.Context, listID, userID string, isAdmin bool) ([]*Share, error) {
	if _, err := s.effectiveRole(ctx, listID, userID, isAdmin); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, listID)
}

// CheckAccess verifies the user has at least minRole on the list. Used by task
// handlers when a task belongs to a shared list.
func (s *TaskListService) CheckAccess(ctx context.Context, listID, userID string, minRole Role, isAdmin bool) error {
	role, err := s.effectiveRole(ctx, listID, userID, isAdmin)
	if err != nil {
		return err
	}
	if !role.AtLeast(minRole) {
		return ErrTaskListAccessDenied
	}
	return nil
}

func (s *TaskListService) UpdateTaskList(ctx context.Context, id string, name, description *string) (*TaskList, error) {
	list, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := list.Update(name, description); err != nil {
		return nil, fmt.Errorf("update task list: %w", err)
	}

	if err := s.repo.Update(ctx, list); err != nil {
		return nil, fmt.Errorf("persist task list update: %w", err)
	}

	log.Info().Str("list_id", list.ID).Msg("task list updated")
	return list, nil
}

func (s *TaskListService) DeleteTaskList(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("list_id", id).Msg("task list deleted")
	return nil
}
