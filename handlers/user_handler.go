package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type UserHandler struct {
	DB *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var row userRow
	err := h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL,
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

	writeJSON(w, http.StatusOK, rowToJSON(row))
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

	_, err := h.DB.Exec(`
		UPDATE users SET name = ?, sex = ?, birth_date = ?, weight_kg = ?, height_cm = ?, language = ?, is_coach = ?, onboarding_completed = ?, updated_at = NOW() WHERE id = ?
	`, req.Name, req.Sex, req.BirthDate, req.WeightKg, req.HeightCm, req.Language, req.IsCoach, req.OnboardingCompleted, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	var row userRow
	err = h.DB.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch updated profile")
		return
	}

	writeJSON(w, http.StatusOK, rowToJSON(row))
}
