package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type AchievementHandler struct {
	DB           *sql.DB
	Notification *NotificationHandler
}

func NewAchievementHandler(db *sql.DB, nh *NotificationHandler) *AchievementHandler {
	return &AchievementHandler{DB: db, Notification: nh}
}

func (h *AchievementHandler) ListMyAchievements(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), COALESCE(extra_info, ''),
			image_file_id, is_public, is_verified, COALESCE(rejection_reason, ''),
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? ORDER BY event_date DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch achievements")
		return
	}
	defer rows.Close()

	achievements := []models.CoachAchievement{}
	for rows.Next() {
		var a models.CoachAchievement
		var verifiedAt sql.NullString
		if err := rows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.ExtraInfo,
			&a.ImageFileID, &a.IsPublic, &a.IsVerified, &a.RejectionReason,
			&a.VerifiedBy, &verifiedAt, &a.CreatedAt); err != nil {
			logErr("scan achievement row", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
		a.EventDate = truncateDate(a.EventDate)
		h.populateImageURL(&a)
		achievements = append(achievements, a)
	}

	writeJSON(w, http.StatusOK, achievements)
}

func (h *AchievementHandler) CreateAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var isCoach bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach); err != nil || !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
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

	result, err := h.DB.Exec(`
		INSERT INTO coach_achievements (coach_id, event_name, event_date, distance_km, result_time, position, extra_info, image_file_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID)
	if err != nil {
		log.Printf("ERROR creating achievement: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create achievement")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for achievement", err)
	}

	h.notifyAdminsNewAchievement(userID, id, req.EventName)

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

	var isVerified bool
	var rejectionReason sql.NullString
	err = h.DB.QueryRow("SELECT is_verified, rejection_reason FROM coach_achievements WHERE id = ? AND coach_id = ?", achID, userID).Scan(&isVerified, &rejectionReason)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Achievement not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch achievement")
		return
	}
	if isVerified {
		writeError(w, http.StatusBadRequest, "Cannot edit a verified achievement")
		return
	}
	// Only rejected achievements can be edited
	if !rejectionReason.Valid || rejectionReason.String == "" {
		writeError(w, http.StatusBadRequest, "Only rejected achievements can be edited")
		return
	}

	var req models.UpdateAchievementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Clear rejection_reason to resubmit as pending
	_, err = h.DB.Exec(`
		UPDATE coach_achievements SET event_name = ?, event_date = ?, distance_km = ?, result_time = ?,
			position = ?, extra_info = ?, image_file_id = ?, rejection_reason = NULL
		WHERE id = ? AND coach_id = ?
	`, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID, achID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update achievement")
		return
	}

	// Re-notify admins about the resubmitted achievement
	h.notifyAdminsNewAchievement(userID, achID, req.EventName)

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

	// Only allow deleting rejected achievements
	result, err := h.DB.Exec("DELETE FROM coach_achievements WHERE id = ? AND coach_id = ? AND rejection_reason IS NOT NULL", achID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete achievement")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found or cannot be deleted")
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

	// Path: /api/coach/achievements/{id}/visibility
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

	result, err := h.DB.Exec("UPDATE coach_achievements SET is_public = ? WHERE id = ? AND coach_id = ?", req.IsPublic, achID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update visibility")
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "Visibility updated", "is_public": req.IsPublic})
}

func (h *AchievementHandler) populateImageURL(a *models.CoachAchievement) {
	if a.ImageFileID == nil {
		return
	}
	var uuid string
	if err := h.DB.QueryRow("SELECT uuid FROM files WHERE id = ?", *a.ImageFileID).Scan(&uuid); err != nil {
		return
	}
	a.ImageURL = "/api/files/" + uuid + "/download"
}

// notifyAdminsNewAchievement sends a notification to all admins about a new/resubmitted achievement.
func (h *AchievementHandler) notifyAdminsNewAchievement(coachUserID, achievementID int64, eventName string) {
	var coachName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", coachUserID).Scan(&coachName); err != nil {
		logErr("fetch coach name for achievement notification", err)
	}
	adminRows, err := h.DB.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		logErr("fetch admin users for achievement notification", err)
		return
	}
	defer adminRows.Close()
	for adminRows.Next() {
		var adminID int64
		if err := adminRows.Scan(&adminID); err != nil {
			continue
		}
		meta := map[string]interface{}{
			"achievement_id": achievementID,
			"event_name":     eventName,
			"coach_name":     coachName,
		}
		h.Notification.CreateNotification(adminID, "achievement_pending",
			"notif_achievement_pending_title", "notif_achievement_pending_body",
			meta, nil)
	}
}
