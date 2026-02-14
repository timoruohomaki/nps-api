package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
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

	_, err := h.db.Collection("feedback").InsertOne(r.Context(), fb)
	if err != nil {
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
