package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type RatingHandler struct {
	svc *services.RatingService
}

func NewRatingHandler(svc *services.RatingService) *RatingHandler {
	return &RatingHandler{svc: svc}
}

// UpsertRating handles POST /api/coaches/{id}/ratings
func (h *RatingHandler) UpsertRating(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/ratings")
	coachID, err := extractID(path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	var req models.UpsertRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = h.svc.Upsert(coachID, userID, req)
	if err == services.ErrNotStudent {
		writeError(w, http.StatusForbidden, "You are not a student of this coach")
		return
	}
	if err == services.ErrInvalidRating {
		writeError(w, http.StatusBadRequest, "Rating must be between 1 and 10")
		return
	}
	if err != nil {
		writeAppError(w, apperr.New(http.StatusInternalServerError, "RatingHandler.UpsertRating", apperr.RATING_001, "Failed to save rating", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Rating saved"})
}

// GetRatings handles GET /api/coaches/{id}/ratings
func (h *RatingHandler) GetRatings(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/ratings")
	coachID, err := extractID(path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	ratings, err := h.svc.List(coachID)
	if err != nil {
		handleServiceErr(w, err, "RatingHandler.GetRatings", apperr.RATING_002, "Failed to fetch ratings")
		return
	}

	writeJSON(w, http.StatusOK, ratings)
}
