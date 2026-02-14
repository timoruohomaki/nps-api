package handler

import (
	"net/http"

	"github.com/idefinity/nps-api/internal/db"
)

// RegisterRoutes sets up all HTTP routes under the /nps prefix.
func RegisterRoutes(database *db.Database) *http.ServeMux {
	mux := http.NewServeMux()
	feedback := NewFeedbackHandler(database)

	mux.HandleFunc("GET /nps/health", HealthCheck)
	mux.HandleFunc("POST /nps/api/v1/feedback", feedback.Submit)

	return mux
}
