package middleware

import (
	"crypto/subtle"
	"net/http"
)

// APIKeyAuth returns middleware that requires a valid X-API-Key header.
//
// If keys is empty, authentication is disabled and every request passes — the
// service stays backward-compatible until keys are provisioned on both the server
// and the clients (the caller logs this state at startup). When keys are set, a
// request must carry an X-API-Key that matches one of them, or it gets 401.
func APIKeyAuth(keys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(keys) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if matchesAny(r.Header.Get("X-API-Key"), keys) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid or missing API key"}`))
		})
	}
}

// RequireAPIKey returns middleware that mandates a valid X-API-Key. Unlike
// APIKeyAuth it is fail-closed: if keys is empty the endpoint is disabled and
// returns 503, so a sensitive endpoint (e.g. one that serves decrypted PII) is
// never exposed without authentication by misconfiguration.
func RequireAPIKey(keys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(keys) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error":"endpoint disabled: no consumer API key configured"}`))
				return
			}
			if !matchesAny(r.Header.Get("X-API-Key"), keys) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid or missing API key"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// matchesAny reports whether provided equals any configured key, using a
// constant-time compare and always checking every key so timing does not reveal
// which key matched or how far the comparison got. An empty provided key never
// matches (keys are non-empty by construction in config).
func matchesAny(provided string, keys []string) bool {
	if provided == "" {
		return false
	}
	match := 0
	for _, k := range keys {
		match |= subtle.ConstantTimeCompare([]byte(provided), []byte(k))
	}
	return match == 1
}
