package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type AchievementHandler struct {
	DB *sql.DB
}

func NewAchievementHandler(db *sql.DB) *AchievementHandler {
	return &AchievementHandler{DB: db}
}

func (h *AchievementHandler) ListMyAchievements(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), is_verified,
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
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.IsVerified,
			&a.VerifiedBy, &verifiedAt, &a.CreatedAt); err != nil {
			logErr("scan achievement row", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
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
		INSERT INTO coach_achievements (coach_id, event_name, event_date, distance_km, result_time, position)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position)
	if err != nil {
		log.Printf("ERROR creating achievement: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create achievement")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for achievement", err)
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

	var isVerified bool
	err = h.DB.QueryRow("SELECT is_verified FROM coach_achievements WHERE id = ? AND coach_id = ?", achID, userID).Scan(&isVerified)
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

	var req models.UpdateAchievementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err = h.DB.Exec(`
		UPDATE coach_achievements SET event_name = ?, event_date = ?, distance_km = ?, result_time = ?, position = ?
		WHERE id = ? AND coach_id = ?
	`, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, achID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update achievement")
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

	result, err := h.DB.Exec("DELETE FROM coach_achievements WHERE id = ? AND coach_id = ?", achID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete achievement")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement deleted"})
}
