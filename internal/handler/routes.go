package handler

import (
	"net/http"

	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/middleware"
)

// RegisterRoutes sets up all HTTP routes under the /nps prefix.
//
//   - health is always open (monitoring + Nginx upstream check)
//   - POST feedback is gated by apiKeys (empty => open, backward-compatible)
//   - GET feedback (analytics query) returns decrypted PII, so it is gated by
//     readKeys and is fail-closed: with no readKeys it returns 503
func RegisterRoutes(database *db.Database, apiKeys, readKeys []string) *http.ServeMux {
	mux := http.NewServeMux()
	feedback := NewFeedbackHandler(database)
	writeAuth := middleware.APIKeyAuth(apiKeys)
	readAuth := middleware.RequireAPIKey(readKeys)

	mux.HandleFunc("GET /nps/health", HealthCheck)
	mux.Handle("POST /nps/api/v1/feedback", writeAuth(http.HandlerFunc(feedback.Submit)))
	mux.Handle("GET /nps/api/v1/feedback", readAuth(http.HandlerFunc(feedback.List)))

	return mux
}
