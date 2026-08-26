package httpx

import (
	"net/http"
)

// CorsMiddleware adds CORS headers when the request Origin is in the configured
// allow-list (see CORS_ALLOWED_ORIGINS).
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
