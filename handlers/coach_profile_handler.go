package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type CoachProfileHandler struct {
	svc *services.CoachProfileService
}

func NewCoachProfileHandler(svc *services.CoachProfileService) *CoachProfileHandler {
	return &CoachProfileHandler{svc: svc}
}

// UpdateCoachProfile handles PUT /api/coach/profile
func (h *CoachProfileHandler) UpdateCoachProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateCoachProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.svc.UpdateProfile(userID, req)
	if err == services.ErrNotCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		log.Printf("ERROR updating coach profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update coach profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Coach profile updated"})
}

// ListCoaches handles GET /api/coaches
func (h *CoachProfileHandler) ListCoaches(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		search = search[:100]
	}
	locality := r.URL.Query().Get("locality")
	if len(locality) > 100 {
		locality = locality[:100]
	}
	level := r.URL.Query().Get("level")
	sortBy := r.URL.Query().Get("sort")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 12
	}
	offset := (page - 1) * limit

	coaches, total, err := h.svc.ListCoaches(search, locality, level, sortBy, limit, offset)
	if err != nil {
		log.Printf("ERROR listing coaches: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coaches")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  coaches,
		"total": total,
	})
}

// GetCoachProfile handles GET /api/coaches/{id}
func (h *CoachProfileHandler) GetCoachProfile(w http.ResponseWriter, r *http.Request) {
	coachID, err := extractID(r.URL.Path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	profile, err := h.svc.GetCoachProfile(coachID, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Coach not found")
		return
	}
	if err != nil {
		log.Printf("ERROR fetching coach profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coach profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
