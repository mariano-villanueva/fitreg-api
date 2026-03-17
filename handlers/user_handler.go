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

type UserHandler struct {
	DB *sql.DB
	NH *NotificationHandler
}

func NewUserHandler(db *sql.DB, nh *NotificationHandler) *UserHandler {
	return &UserHandler{DB: db, NH: nh}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var row userRow
	err := h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	u := rowToJSON(row)
	var hasCoach bool
	if err := h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coach_students WHERE student_id = ? AND status = 'active')", userID).Scan(&hasCoach); err != nil {
		logErr("check has coach for profile", err)
	}
	u.HasCoach = hasCoach
	if hasCoach {
		fillCoachInfoDB(h.DB, userID, &u)
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var birthDate interface{} = req.BirthDate
	if req.BirthDate == "" {
		birthDate = nil
	}
	var sex interface{} = req.Sex
	if req.Sex == "" {
		sex = nil
	}

	_, err := h.DB.Exec(`
		UPDATE users SET name = ?, sex = ?, birth_date = ?, weight_kg = ?, height_cm = ?, language = ?, onboarding_completed = ?, updated_at = NOW() WHERE id = ?
	`, req.Name, sex, birthDate, req.WeightKg, req.HeightCm, req.Language, req.OnboardingCompleted, userID)
	if err != nil {
		log.Printf("ERROR UpdateProfile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	var row userRow
	err = h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch updated profile")
		return
	}

	writeJSON(w, http.StatusOK, rowToJSON(row))
}

func (h *UserHandler) RequestCoach(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Locality string   `json:"locality"`
		Level    []string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Level) == 0 {
		writeError(w, http.StatusBadRequest, "At least one level is required")
		return
	}

	// Check if already a coach
	var isCoach bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach); err != nil {
		logErr("check is coach for request", err)
	}
	if isCoach {
		writeError(w, http.StatusConflict, "User is already a coach")
		return
	}

	// Check if there's already a pending coach request
	var pendingCount int
	if err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, userID).Scan(&pendingCount); err != nil {
		logErr("check pending coach request count", err)
	}
	if pendingCount > 0 {
		writeError(w, http.StatusConflict, "Coach request already pending")
		return
	}

	// Save locality and level on user (levels as comma-separated string)
	levelStr := strings.Join(req.Level, ",")
	if _, err := h.DB.Exec("UPDATE users SET coach_locality = ?, coach_level = ?, updated_at = NOW() WHERE id = ?",
		req.Locality, levelStr, userID); err != nil {
		logErr("update user coach locality and level", err)
	}

	// Get requester name
	var requesterName, requesterAvatar string
	if err := h.DB.QueryRow("SELECT COALESCE(name, ''), COALESCE(custom_avatar, '') FROM users WHERE id = ?", userID).Scan(&requesterName, &requesterAvatar); err != nil {
		logErr("fetch requester name for coach request", err)
	}

	// Get all admin users
	rows, err := h.DB.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch admins")
		return
	}
	defer rows.Close()

	var adminIDs []int64
	for rows.Next() {
		var adminID int64
		if err := rows.Scan(&adminID); err == nil {
			adminIDs = append(adminIDs, adminID)
		}
	}

	if len(adminIDs) == 0 {
		log.Println("WARNING: No admin users found for coach request notification")
		writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
		return
	}

	// Create notification for each admin
	meta := map[string]interface{}{
		"requester_id":     userID,
		"requester_name":   requesterName,
		"requester_avatar": requesterAvatar,
		"locality":         req.Locality,
		"level":            req.Level,
	}
	actions := []models.NotificationAction{
		{Key: "approve", Label: "notif_coach_request_approve", Style: "primary"},
		{Key: "reject", Label: "notif_coach_request_reject", Style: "danger"},
	}

	for _, adminID := range adminIDs {
		h.NH.CreateNotification(adminID, "coach_request",
			"notif_coach_request_title", "notif_coach_request_body",
			meta, actions)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
}

func (h *UserHandler) GetCoachRequestStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if already a coach
	var isCoach bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach); err != nil {
		logErr("check is coach for request status", err)
	}
	if isCoach {
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
		return
	}

	// Check if there's a pending coach request
	var pendingCount int
	if err := h.DB.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, userID).Scan(&pendingCount); err != nil {
		logErr("check pending coach request status", err)
	}

	if pendingCount > 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "none"})
}

const maxAvatarSize = 500 * 1024 // 500KB base64

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	if !strings.HasPrefix(req.Image, "data:image/") {
		writeError(w, http.StatusBadRequest, "image must be a base64 data URI")
		return
	}

	if len(req.Image) > maxAvatarSize {
		writeError(w, http.StatusBadRequest, "image too large (max 500KB)")
		return
	}

	if _, err := h.DB.Exec("UPDATE users SET custom_avatar = ?, updated_at = NOW() WHERE id = ?", req.Image, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar updated"})
}

func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if _, err := h.DB.Exec("UPDATE users SET custom_avatar = NULL, updated_at = NOW() WHERE id = ?", userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar removed"})
}

// TEMPORARY: These will be removed in Task 4 when UserHandler is migrated to use UserService.
// They are duplicated here to keep the build passing after Task 3 removed them from auth_handler.go.

type userRow = models.UserRow // alias to the new models type

func rowToJSON(row models.UserRow) models.UserProfile {
	u := models.UserProfile{
		ID:        row.ID,
		GoogleID:  row.GoogleID,
		Email:     row.Email,
		Name:      row.Name,
		Language:  "es",
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.CustomAvatar.Valid {
		u.CustomAvatar = row.CustomAvatar.String
		u.AvatarURL = row.CustomAvatar.String
	}
	if row.Sex.Valid {
		u.Sex = row.Sex.String
	}
	if row.BirthDate.Valid {
		bd := truncateDate(row.BirthDate.String)
		u.BirthDate = bd
		u.Age = models.CalculateAge(bd)
	}
	if row.WeightKg.Valid {
		u.WeightKg = row.WeightKg.Float64
	}
	if row.HeightCm.Valid {
		u.HeightCm = int(row.HeightCm.Int64)
	}
	if row.Language.Valid {
		u.Language = row.Language.String
	}
	if row.IsCoach.Valid {
		u.IsCoach = row.IsCoach.Bool
	}
	if row.IsAdmin.Valid {
		u.IsAdmin = row.IsAdmin.Bool
	}
	if row.CoachDescription.Valid {
		u.CoachDescription = row.CoachDescription.String
	}
	if row.CoachPublic.Valid {
		u.CoachPublic = row.CoachPublic.Bool
	}
	if row.OnboardingCompleted.Valid {
		u.OnboardingCompleted = row.OnboardingCompleted.Bool
	}
	return u
}

func fillCoachInfoDB(db *sql.DB, studentID int64, u *models.UserProfile) {
	var coachID int64
	var coachName string
	var coachAvatar sql.NullString
	err := db.QueryRow(`
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '')
		FROM coach_students cs
		JOIN users u ON u.id = cs.coach_id
		WHERE cs.student_id = ? AND cs.status = 'active'
		LIMIT 1
	`, studentID).Scan(&coachID, &coachName, &coachAvatar)
	if err != nil {
		logErr("fetch coach info for user", err)
		return
	}
	u.CoachID = coachID
	u.CoachName = coachName
	if coachAvatar.Valid {
		u.CoachAvatar = coachAvatar.String
	}
}
