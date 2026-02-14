package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/idefinity/nps-api/internal/model"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type FeedbackHandler struct {
	collection *mongo.Collection
}

func NewFeedbackHandler(db *mongo.Database) *FeedbackHandler {
	return &FeedbackHandler{
		collection: db.Collection("feedback"),
	}
}

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

	_, err := h.collection.InsertOne(r.Context(), fb)
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

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
