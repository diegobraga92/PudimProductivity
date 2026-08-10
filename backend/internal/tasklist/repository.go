package tasklist

import (
	"context"
	"errors"
)

var (
	ErrTaskListNotFound = errors.New("task list not found")
	// ErrTaskListAccessDenied is returned when a user is not a member
	// (owner or share) of the task list, or their role is insufficient.
	ErrTaskListAccessDenied = errors.New("access denied to task list")
	// ErrShareNotFound is returned when removing a share that does not exist.
	ErrShareNotFound = errors.New("share not found")
	// ErrShareExists is returned when sharing a list with a user who is
	// already a member (or is the owner).
	ErrShareExists = errors.New("share already exists")
)

type TaskListRepository interface {
	// Create persists a new task list.
	Create(ctx context.Context, list *TaskList) error

	// GetByID retrieves a task list by its ID.
	// Returns ErrTaskListNotFound if the list does not exist.
	GetByID(ctx context.Context, id string) (*TaskList, error)

	// List returns all task lists.
	List(ctx context.Context) ([]*TaskList, error)

	// Update persists changes to an existing task list.
	// Returns ErrTaskListNotFound if the list does not exist.
	Update(ctx context.Context, list *TaskList) error

	// Delete removes a task list by its ID.
	// Returns ErrTaskListNotFound if the list does not exist.
	Delete(ctx context.Context, id string) error

	// ShareRepository
	ShareRepository
}

// ShareRepository is the Phase 8 collaboration surface. It is embedded in
// TaskListRepository so a single Postgres implementation satisfies both.
type ShareRepository interface {
	// GetMemberRole returns the effective role for a user on a list:
	// RoleOwner if they own it, the share role if they were invited, or
	// ErrTaskListAccessDenied if they have no access. Admins bypass by passing
	// RoleOwner via the higher-level service.
	GetMemberRole(ctx context.Context, listID, userID string) (Role, error)

	// CreateShare grants a role to another user.
	// Returns ErrShareExists if a share already exists.
	CreateShare(ctx context.Context, share *Share) error

	// DeleteShare revokes a share.
	// Returns ErrShareNotFound if no share exists.
	DeleteShare(ctx context.Context, listID, userID string) error

	// ListShares returns all non-owner members of a list.
	ListShares(ctx context.Context, listID string) ([]*Share, error)

	// ListListsForUser returns the task lists the user owns or is a member of
	// (owner rows + share rows). Includes owner_id for role display.
	ListListsForUser(ctx context.Context, userID string) ([]*TaskList, error)

	// ListListIDsForUser returns just the IDs of the lists a user owns or is a
	// member of. Used by the sync hub for presence + event scoping.
	ListListIDsForUser(ctx context.Context, userID string) ([]string, error)

	// ListMemberUserIDs returns the user IDs who can see a list (owner +
	// shared_with). Used by the sync hub for presence broadcasting.
	ListMemberUserIDs(ctx context.Context, listID string) ([]string, error)
}
