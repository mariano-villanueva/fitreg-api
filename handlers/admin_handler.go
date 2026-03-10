package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
)

type AdminHandler struct {
	DB *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

func (h *AdminHandler) requireAdmin(userID int64) bool {
	var isAdmin bool
	err := h.DB.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return err == nil && isAdmin
}

func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	var totalUsers, totalCoaches, totalRatings, pendingAchievements int
	h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_coach = TRUE").Scan(&totalCoaches)
	h.DB.QueryRow("SELECT COUNT(*) FROM coach_ratings").Scan(&totalRatings)
	h.DB.QueryRow("SELECT COUNT(*) FROM coach_achievements WHERE is_verified = FALSE").Scan(&pendingAchievements)

	writeJSON(w, http.StatusOK, map[string]int{
		"total_users":          totalUsers,
		"total_coaches":        totalCoaches,
		"total_ratings":        totalRatings,
		"pending_achievements": pendingAchievements,
	})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, email, name, COALESCE(avatar_url, '') as avatar_url,
			COALESCE(is_coach, FALSE), COALESCE(is_admin, FALSE), created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer rows.Close()

	type AdminUser struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		IsCoach   bool   `json:"is_coach"`
		IsAdmin   bool   `json:"is_admin"`
		CreatedAt string `json:"created_at"`
	}

	users := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL,
			&u.IsCoach, &u.IsAdmin, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	targetID, err := extractID(r.URL.Path, "/api/admin/users/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		IsCoach *bool `json:"is_coach"`
		IsAdmin *bool `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.IsCoach != nil {
		h.DB.Exec("UPDATE users SET is_coach = ?, updated_at = NOW() WHERE id = ?", *req.IsCoach, targetID)
	}
	if req.IsAdmin != nil {
		h.DB.Exec("UPDATE users SET is_admin = ?, updated_at = NOW() WHERE id = ?", *req.IsAdmin, targetID)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User updated"})
}

func (h *AdminHandler) PendingAchievements(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	rows, err := h.DB.Query(`
		SELECT ca.id, ca.coach_id, ca.event_name, ca.event_date,
			COALESCE(ca.distance_km, 0), COALESCE(ca.result_time, ''),
			COALESCE(ca.position, 0), ca.created_at, u.name as coach_name
		FROM coach_achievements ca
		JOIN users u ON u.id = ca.coach_id
		WHERE ca.is_verified = FALSE
		ORDER BY ca.created_at ASC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch achievements")
		return
	}
	defer rows.Close()

	type PendingAchievement struct {
		ID         int64   `json:"id"`
		CoachID    int64   `json:"coach_id"`
		EventName  string  `json:"event_name"`
		EventDate  string  `json:"event_date"`
		DistanceKm float64 `json:"distance_km"`
		ResultTime string  `json:"result_time"`
		Position   int     `json:"position"`
		CreatedAt  string  `json:"created_at"`
		CoachName  string  `json:"coach_name"`
	}

	achievements := []PendingAchievement{}
	for rows.Next() {
		var a PendingAchievement
		if err := rows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.CreatedAt, &a.CoachName); err != nil {
			continue
		}
		achievements = append(achievements, a)
	}

	writeJSON(w, http.StatusOK, achievements)
}

func (h *AdminHandler) VerifyAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/verify")
	achID, err := extractID(path, "/api/admin/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}

	result, err := h.DB.Exec(`
		UPDATE coach_achievements SET is_verified = TRUE, verified_by = ?, verified_at = NOW()
		WHERE id = ? AND is_verified = FALSE
	`, userID, achID)
	if err != nil {
		log.Printf("ERROR verifying achievement: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to verify achievement")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found or already verified")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement verified"})
}

func (h *AdminHandler) RejectAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if !h.requireAdmin(userID) {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/reject")
	achID, err := extractID(path, "/api/admin/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}

	result, err := h.DB.Exec("DELETE FROM coach_achievements WHERE id = ? AND is_verified = FALSE", achID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reject achievement")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found or already verified")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement rejected"})
}
