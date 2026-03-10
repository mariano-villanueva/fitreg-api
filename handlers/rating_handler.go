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

type RatingHandler struct {
	DB *sql.DB
}

func NewRatingHandler(db *sql.DB) *RatingHandler {
	return &RatingHandler{DB: db}
}

// UpsertRating handles POST /api/coaches/{id}/ratings
func (h *RatingHandler) UpsertRating(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Path: /api/coaches/{id}/ratings — remove /ratings to get coach ID
	path := strings.TrimSuffix(r.URL.Path, "/ratings")
	coachID, err := extractID(path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ?", coachID, userID).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusForbidden, "You are not a student of this coach")
		return
	}

	var req models.UpsertRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Rating < 1 || req.Rating > 10 {
		writeError(w, http.StatusBadRequest, "Rating must be between 1 and 10")
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO coach_ratings (coach_id, student_id, rating, comment)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE rating = VALUES(rating), comment = VALUES(comment), updated_at = NOW()
	`, coachID, userID, req.Rating, req.Comment)
	if err != nil {
		log.Printf("ERROR upserting rating: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save rating")
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

	rows, err := h.DB.Query(`
		SELECT cr.id, cr.coach_id, cr.student_id, cr.rating, COALESCE(cr.comment, ''),
			u.name as student_name, cr.created_at, cr.updated_at
		FROM coach_ratings cr
		JOIN users u ON u.id = cr.student_id
		WHERE cr.coach_id = ? ORDER BY cr.updated_at DESC
	`, coachID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch ratings")
		return
	}
	defer rows.Close()

	ratings := []models.CoachRating{}
	for rows.Next() {
		var rt models.CoachRating
		if err := rows.Scan(&rt.ID, &rt.CoachID, &rt.StudentID, &rt.Rating,
			&rt.Comment, &rt.StudentName, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			continue
		}
		ratings = append(ratings, rt)
	}

	writeJSON(w, http.StatusOK, ratings)
}
