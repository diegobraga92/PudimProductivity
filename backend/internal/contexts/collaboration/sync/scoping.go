package sync

import (
	"context"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// MembershipResolver resolves which task lists a user can access. It is used by
// the hub to (a) scope event dispatch to list members and (b) compute presence.
//
// Phase 8: the postgres package provides a Postgres-backed implementation.
type MembershipResolver interface {
	// ListIDsForUser returns the IDs of the task lists the user owns or is a
	// member of (via task_list_shares). role is the user's application role
	// ("admin" sees all lists).
	ListIDsForUser(ctx context.Context, userID, role string) ([]string, error)
}

// targetsFor extracts the list IDs an event pertains to. The second return
// value is true when dispatch must be restricted to members of those lists;
// when false the event is broadcast to every connected client (legacy events
// and non-list-scoped events such as presence).
func targetsFor(event eventbus.Event) ([]string, bool) {
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
