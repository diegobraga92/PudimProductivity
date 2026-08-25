package tasklist

import (
	"fmt"
	"time"
)

// Role is a member's permission level on a shared task list.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Valid reports whether r is a known share role.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// AtLeast reports whether r grants at least the permission level of min.
// Ordering: viewer < editor < owner.
func (r Role) AtLeast(min Role) bool {
	rank := func(role Role) int {
		switch role {
		case RoleOwner:
			return 3
		case RoleEditor:
			return 2
		default:
			return 1
		}
	}
	return rank(r) >= rank(min)
}

// Share is a task-list membership granted to another user. The owner is stored
// on TaskList.OwnerID rather than as a share row.
type Share struct {
	ListID     string
	SharedWith string
	Role       Role
	CreatedAt  time.Time
}

// NewShare validates and builds a Share.
func NewShare(listID, sharedWith string, role Role) (*Share, error) {
	if listID == "" {
		return nil, fmt.Errorf("list id cannot be empty")
	}
	if sharedWith == "" {
		return nil, fmt.Errorf("shared_with cannot be empty")
	}
	if role != RoleEditor && role != RoleViewer {
		return nil, fmt.Errorf("share role must be editor or viewer")
	}
	return &Share{
		ListID:     listID,
		SharedWith: sharedWith,
		Role:       role,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

type TaskList struct {
	ID          string
	Name        string
	Description string
	OwnerID     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewTaskList(id, name string) (*TaskList, error) {
	if id == "" {
		return nil, fmt.Errorf("task list id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("task list name cannot be empty")
	}

	now := time.Now().UTC()
	return &TaskList{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (l *TaskList) Update(name, description *string) error {
	if name != nil {
		if *name == "" {
			return fmt.Errorf("task list name cannot be empty")
		}
		l.Name = *name
	}
	if description != nil {
		l.Description = *description
	}
	l.UpdatedAt = time.Now().UTC()
	return nil
}
