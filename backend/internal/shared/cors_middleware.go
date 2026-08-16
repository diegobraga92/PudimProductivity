package shared

import (
	"net/http"
	"strings"
)

// CorsMiddleware adds CORS headers when the request Origin is in the configured
// allow-list (see CORS_ALLOWED_ORIGINS). When no origins are configured it is a
// no-op, preserving the same-origin behavior used by the web (nginx) and Vite
// dev deployments. Allowed origins are matched exactly (e.g. "app://bundle" for
// the Electron desktop app).
//
// Requests without an Origin header (curl, internal services) and requests from
// non-allowed origins pass through unchanged — we never add headers for them.
// WebSocket upgrades are normal GET requests with an Upgrade header: they pass
// through untouched (we only set response headers), and the sync hub performs
// the actual hijack further down the chain.
func CorsMiddleware(allowedOrigins map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allowedOrigins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" || !allowedOrigins[origin] {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			// The dev identity headers plus the future Authorization credential.
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID, X-User-Role, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")

			// Answer CORS preflights without invoking the rest of the chain
			// (the browser sends OPTIONS with Access-Control-Request-Method).
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ParseAllowedOrigins splits a comma-separated origin list into a lookup set.
// Whitespace is trimmed and empty entries are ignored.
func ParseAllowedOrigins(raw string) map[string]bool {
	set := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			set[entry] = true
		}
	}
	return set
}
