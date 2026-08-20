package httpx

import (
	"context"
	"net/http"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyUserRole contextKey = "user_role"
)

func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return v
	}
	return ""
}

func GetUserRole(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUserRole).(string); ok {
		return v
	}
	return ""
}

// TODO: In development, this reads X-User-ID and X-User-Role headers.
// TODO: In production, this would validate JWT tokens or session cookies.
//
// When a request carries no identity headers, it defaults to the dev user
// ("dev-user"/"user") so that read and write requests share the same identity.
// The web client always sends these headers on mutations (see
// web/src/api/client.ts DEV_USER_ID/DEV_USER_ROLE), but reads omit them; a
// default matching the dev user keeps ownership filtering (Phase 8) consistent
// between the two. Production will replace this with JWT/session validation.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		userRole := r.Header.Get("X-User-Role")

		if userID == "" {
			userID = "dev-user"
			userRole = "user"
		}

		ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyUserRole, userRole)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, role := range roles {
		roleSet[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := GetUserRole(r.Context())
			if !roleSet[userRole] {
				WriteError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
