package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/idefinity/nps-api/internal/db"
	"github.com/idefinity/nps-api/internal/model"
)

// FeedbackHandler handles NPS feedback submissions.
type FeedbackHandler struct {
	db *db.Database
}

// NewFeedbackHandler creates a handler backed by the given database.
func NewFeedbackHandler(database *db.Database) *FeedbackHandler {
	return &FeedbackHandler{db: database}
}

// Submit handles POST requests to store NPS feedback.
func (h *FeedbackHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var fb model.Feedback
	if err := json.NewDecoder(r.Body).Decode(&fb); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON payload",
		})
		return
	}

	if err := fb.Validate(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": err.Error(),
		})
		return
	}

	fb.ReceivedAt = time.Now().UTC()

	if err := h.db.InsertFeedback(r.Context(), &fb); err != nil {
		slog.Error("failed to insert feedback", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to store feedback",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "ok",
	})
}

// List handles GET requests that query stored feedback for analytics. An
// optional ?year=YYYY filters by receipt year. The response is a JSON array of
// feedback with the PII fields decrypted — callers are gated by a consumer API
// key (see routes.go).
func (h *FeedbackHandler) List(w http.ResponseWriter, r *http.Request) {
	year := 0
	if y := r.URL.Query().Get("year"); y != "" {
		parsed, err := strconv.Atoi(y)
		if err != nil || parsed < 2000 || parsed > 2200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "year must be a 4-digit year between 2000 and 2200",
			})
			return
		}
		year = parsed
	}

	items, err := h.db.ListFeedback(r.Context(), year)
	if err != nil {
		slog.Error("failed to list feedback", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read feedback",
		})
		return
	}

	// Always return an array, never null, for an empty result.
	if items == nil {
		items = []model.Feedback{}
	}
	writeJSON(w, http.StatusOK, items)
}
