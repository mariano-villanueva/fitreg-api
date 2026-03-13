package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
)

type AdminHandler struct {
	DB           *sql.DB
	Notification *NotificationHandler
}

func NewAdminHandler(db *sql.DB, nh *NotificationHandler) *AdminHandler {
	return &AdminHandler{DB: db, Notification: nh}
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
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers); err != nil {
		logErr("count total users", err)
	}
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_coach = TRUE").Scan(&totalCoaches); err != nil {
		logErr("count total coaches", err)
	}
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM coach_ratings").Scan(&totalRatings); err != nil {
		logErr("count total ratings", err)
	}
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM coach_achievements WHERE is_verified = FALSE AND rejection_reason IS NULL").Scan(&pendingAchievements); err != nil {
		logErr("count pending achievements", err)
	}

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

	// Parse query params
	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")
	sortCol := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Defaults and clamping
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Whitelist sort column
	allowedSort := map[string]string{
		"name":       "name",
		"email":      "email",
		"created_at": "created_at",
	}
	if _, ok := allowedSort[sortCol]; !ok {
		sortCol = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Build dynamic WHERE
	where := "WHERE 1=1"
	args := []interface{}{}

	if search != "" {
		where += " AND (name LIKE ? OR email LIKE ?)"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}

	switch role {
	case "athlete":
		where += " AND COALESCE(is_coach, FALSE) = FALSE AND COALESCE(is_admin, FALSE) = FALSE"
	case "coach":
		where += " AND COALESCE(is_coach, FALSE) = TRUE"
	case "admin":
		where += " AND COALESCE(is_admin, FALSE) = TRUE"
	}

	// Count total matching users
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM users "+where, countArgs...).Scan(&total); err != nil {
		logErr("count admin users", err)
		writeError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	// Paginated SELECT
	offset := (page - 1) * limit
	args = append(args, limit, offset)
	query := `
		SELECT id, email, COALESCE(name, '') as name,
			COALESCE(custom_avatar, avatar_url, '') as avatar_url,
			COALESCE(is_coach, FALSE), COALESCE(is_admin, FALSE), created_at
		FROM users ` + where + ` ORDER BY ` + sortCol + ` ` + sortOrder + ` LIMIT ? OFFSET ?`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		logErr("list admin users", err)
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
			logErr("scan admin user row", err)
			continue
		}
		users = append(users, u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
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
		if _, err := h.DB.Exec("UPDATE users SET is_coach = ?, updated_at = NOW() WHERE id = ?", *req.IsCoach, targetID); err != nil {
			logErr("admin update user is_coach", err)
		}
	}
	if req.IsAdmin != nil {
		if _, err := h.DB.Exec("UPDATE users SET is_admin = ?, updated_at = NOW() WHERE id = ?", *req.IsAdmin, targetID); err != nil {
			logErr("admin update user is_admin", err)
		}
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
			COALESCE(ca.position, 0), COALESCE(ca.extra_info, ''),
			ca.image_file_id, ca.created_at, u.name as coach_name
		FROM coach_achievements ca
		JOIN users u ON u.id = ca.coach_id
		WHERE ca.is_verified = FALSE AND ca.rejection_reason IS NULL
		ORDER BY ca.created_at ASC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch achievements")
		return
	}
	defer rows.Close()

	type PendingAchievement struct {
		ID          int64   `json:"id"`
		CoachID     int64   `json:"coach_id"`
		EventName   string  `json:"event_name"`
		EventDate   string  `json:"event_date"`
		DistanceKm  float64 `json:"distance_km"`
		ResultTime  string  `json:"result_time"`
		Position    int     `json:"position"`
		ExtraInfo   string  `json:"extra_info"`
		ImageFileID *int64  `json:"image_file_id"`
		ImageURL    string  `json:"image_url,omitempty"`
		CreatedAt   string  `json:"created_at"`
		CoachName   string  `json:"coach_name"`
	}

	achievements := []PendingAchievement{}
	for rows.Next() {
		var a PendingAchievement
		if err := rows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.ExtraInfo,
			&a.ImageFileID, &a.CreatedAt, &a.CoachName); err != nil {
			logErr("scan pending achievement row", err)
			continue
		}
		a.EventDate = truncateDate(a.EventDate)
		if a.ImageFileID != nil {
			var uuid string
			if err := h.DB.QueryRow("SELECT uuid FROM files WHERE id = ?", *a.ImageFileID).Scan(&uuid); err == nil {
				a.ImageURL = "/api/files/" + uuid + "/download"
			}
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
		UPDATE coach_achievements SET is_verified = TRUE, rejection_reason = NULL, verified_by = ?, verified_at = NOW()
		WHERE id = ? AND is_verified = FALSE
	`, userID, achID)
	if err != nil {
		log.Printf("ERROR verifying achievement: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to verify achievement")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for verify achievement", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found or already verified")
		return
	}

	// Notify coach about verified achievement
	var coachID int64
	var eventName string
	if err := h.DB.QueryRow("SELECT coach_id, event_name FROM coach_achievements WHERE id = ?", achID).Scan(&coachID, &eventName); err != nil {
		logErr("fetch achievement for verification notification", err)
	}
	meta := map[string]interface{}{"achievement_id": achID, "event_name": eventName}
	h.Notification.CreateNotification(coachID, "achievement_verified", "notif_achievement_verified_title", "notif_achievement_verified_body", meta, nil)

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

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for backwards compatibility
		req.Reason = ""
	}

	// Fetch achievement info before rejecting
	var coachID int64
	var eventName string
	if err := h.DB.QueryRow("SELECT coach_id, event_name FROM coach_achievements WHERE id = ? AND is_verified = FALSE AND rejection_reason IS NULL", achID).Scan(&coachID, &eventName); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Achievement not found or already processed")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to fetch achievement")
		return
	}

	// Mark as rejected instead of deleting
	result, err := h.DB.Exec("UPDATE coach_achievements SET rejection_reason = ? WHERE id = ? AND is_verified = FALSE", req.Reason, achID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reject achievement")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for reject achievement", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Achievement not found or already processed")
		return
	}

	// Notify coach about rejected achievement
	meta := map[string]interface{}{
		"achievement_id": achID,
		"event_name":     eventName,
		"reason":         req.Reason,
	}
	h.Notification.CreateNotification(coachID, "achievement_rejected",
		"notif_achievement_rejected_title", "notif_achievement_rejected_body",
		meta, nil)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement rejected"})
}
