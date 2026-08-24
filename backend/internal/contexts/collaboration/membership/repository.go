// Package membership wires collaboration infrastructure into the server.
package membership

import "context"

// Repository resolves which task lists a user can access.
type Repository interface {
	// ListIDsForUser returns the IDs of the task lists the user owns or is a member
	ListIDsForUser(ctx context.Context, userID, role string) ([]string, error)
}
