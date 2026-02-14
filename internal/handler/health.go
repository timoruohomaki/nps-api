package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

// HealthResponse is the JSON structure returned by the health endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Sentry    string `json:"sentry"`
}

// HealthCheck responds with service status, timestamp, and Sentry status.
// Pass ?sentry_test=1 to send a test event.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	sentryStatus := "disabled"

	hub := sentry.GetHubFromContext(r.Context())
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	if hub.Client() != nil {
		sentryStatus = "enabled"

		if r.URL.Query().Get("sentry_test") == "1" {
			hub.CaptureMessage("health check test event from nps-api")
			sentryStatus = "test event sent"
		}
	}

	resp := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Sentry:    sentryStatus,
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
