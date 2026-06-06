package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKey returns middleware that requires an X-API-Key header matching one of
// allowedKeys for any request whose path begins with one of requirePrefixes.
// If allowedKeys is empty the middleware is a no-op, preserving the historical
// open-endpoint behavior so existing deployments do not break on upgrade.
// Constant-time comparison is used to avoid leaking key contents via timing.
func APIKey(allowedKeys []string, requirePrefixes []string) func(http.Handler) http.Handler {
	if len(allowedKeys) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	keys := make([][]byte, 0, len(allowedKeys))
	for _, k := range allowedKeys {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, []byte(k))
		}
	}
	if len(keys) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !pathMatchesAny(r.URL.Path, requirePrefixes) {
				next.ServeHTTP(w, r)
				return
			}

			provided := []byte(r.Header.Get("X-API-Key"))
			for _, k := range keys {
				if subtle.ConstantTimeCompare(provided, k) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		})
	}
}

func pathMatchesAny(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
