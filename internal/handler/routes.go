package handler

import (
	"net/http"

	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/middleware"
)

// RegisterRoutes sets up all HTTP routes under the /nps prefix. apiKeys gates the
// feedback submission endpoint (empty => open); the health check is always open
// so monitoring and the Nginx upstream check work without a key.
func RegisterRoutes(database *db.Database, apiKeys []string) *http.ServeMux {
	mux := http.NewServeMux()
	feedback := NewFeedbackHandler(database)
	auth := middleware.APIKeyAuth(apiKeys)

	mux.HandleFunc("GET /nps/health", HealthCheck)
	mux.Handle("POST /nps/api/v1/feedback", auth(http.HandlerFunc(feedback.Submit)))

	return mux
}
