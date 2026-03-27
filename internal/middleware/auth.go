package middleware

import (
	"log/slog"
	"net/http"
)

// AdminAuth returns middleware that validates the X-API-Key header against apiKey.
// If apiKey is empty (dev mode), requests pass through with a warning.
func AdminAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				slog.Warn("ADMIN_API_KEY not set — admin endpoints are unprotected")
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("X-API-Key")
			if key == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing API key")
				return
			}
			if key != apiKey {
				writeAuthError(w, http.StatusForbidden, "invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
