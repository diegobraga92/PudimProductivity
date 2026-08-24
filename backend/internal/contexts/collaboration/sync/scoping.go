package sync

import (
	"context"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// MembershipResolver resolves which task lists a user can access.
type MembershipResolver interface {
	// ListIDsForUser returns the IDs of the task lists the user owns or is a member of.
	ListIDsForUser(ctx context.Context, userID, role string) ([]string, error)
}

func affectedListIDs(event eventbus.Event) ([]string, bool) {
	switch event.Type {
	case eventbus.EventTaskCreated,
		eventbus.EventTaskUpdated,
		eventbus.EventTaskMerged,
		eventbus.EventTaskListShared,
		eventbus.EventTaskListUnshared:
		if m, ok := event.Payload.(map[string]any); ok {
			if lid, ok := m["list_id"].(string); ok && lid != "" {
				return []string{lid}, true
			}
		}
	}
	return nil, false
}
