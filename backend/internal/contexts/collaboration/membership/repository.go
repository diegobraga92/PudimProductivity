// Package membership wires collaboration infrastructure into the server.
package membership

import "context"

// TODO: Add proper users, login and auth to make use of membership

// Repository resolves which task lists a user can access. It backs the sync
// hub's event scoping and presence features.
type Repository interface {
	// ListIDsForUser returns the IDs of the task lists the user owns or is a
	// member of (via task_list_shares). role is the user's application role
	// ("admin" sees every list).
	ListIDsForUser(ctx context.Context, userID, role string) ([]string, error)
}
