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

type AchievementHandler struct {
	svc *services.AchievementService
}

func NewAchievementHandler(svc *services.AchievementService) *AchievementHandler {
	return &AchievementHandler{svc: svc}
}

func (h *AchievementHandler) ListMyAchievements(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	achievements, err := h.svc.ListMy(userID)
	if err != nil {
		handleServiceErr(w, err, "AchievementHandler.ListMyAchievements", apperr.ACHIEVEMENT_001, "Failed to fetch achievements")
		return
	}
	writeJSON(w, http.StatusOK, achievements)
}

func (h *AchievementHandler) CreateAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateAchievementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.EventName == "" || req.EventDate == "" {
		writeError(w, http.StatusBadRequest, "event_name and event_date are required")
		return
	}
	id, err := h.svc.Create(userID, req)
	if err != nil {
		handleServiceErr(w, err, "AchievementHandler.CreateAchievement", apperr.ACHIEVEMENT_002, "Failed to create achievement")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "message": "Achievement created"})
}

func (h *AchievementHandler) UpdateAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	achID, err := extractID(r.URL.Path, "/api/coach/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}
	var req models.UpdateAchievementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.Update(achID, userID, req); err != nil {
		handleServiceErr(w, err, "AchievementHandler.UpdateAchievement", apperr.ACHIEVEMENT_003, "Failed to update achievement")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement updated"})
}

func (h *AchievementHandler) DeleteAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	achID, err := extractID(r.URL.Path, "/api/coach/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}
	if err := h.svc.Delete(achID, userID); err != nil {
		handleServiceErr(w, err, "AchievementHandler.DeleteAchievement", apperr.ACHIEVEMENT_004, "Failed to delete achievement")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement deleted"})
}

func (h *AchievementHandler) ToggleVisibility(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/visibility")
	achID, err := extractID(path, "/api/coach/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}
	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.SetVisibility(achID, userID, req.IsPublic); err != nil {
		handleServiceErr(w, err, "AchievementHandler.ToggleVisibility", apperr.ACHIEVEMENT_005, "Failed to update visibility")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "Visibility updated", "is_public": req.IsPublic})
}
