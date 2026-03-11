package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type CoachProfileHandler struct {
	DB *sql.DB
}

func NewCoachProfileHandler(db *sql.DB) *CoachProfileHandler {
	return &CoachProfileHandler{DB: db}
}

// UpdateCoachProfile handles PUT /api/coach/profile
func (h *CoachProfileHandler) UpdateCoachProfile(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateCoachProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err := h.DB.Exec("UPDATE users SET coach_description = ?, coach_public = ?, updated_at = NOW() WHERE id = ?",
		req.CoachDescription, req.CoachPublic, userID)
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
	locality := r.URL.Query().Get("locality")
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

	where := "WHERE u.is_coach = TRUE AND u.coach_public = TRUE"
	args := []interface{}{}

	if search != "" {
		where += " AND (u.name LIKE ? OR u.coach_description LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if locality != "" {
		where += " AND u.coach_locality LIKE ?"
		args = append(args, "%"+locality+"%")
	}
	if level != "" {
		where += " AND FIND_IN_SET(?, u.coach_level) > 0"
		args = append(args, level)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(DISTINCT u.id) FROM users u " + where
	if err := h.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		logErr("count coaches", err)
	}

	query := `
		SELECT u.id, u.name, COALESCE(u.avatar_url, '') as avatar_url,
			COALESCE(u.coach_description, '') as coach_description,
			COALESCE(u.coach_locality, '') as coach_locality,
			COALESCE(u.coach_level, '') as coach_level,
			COALESCE(AVG(cr.rating), 0) as avg_rating,
			COUNT(cr.id) as rating_count,
			(SELECT COUNT(*) FROM coach_achievements ca WHERE ca.coach_id = u.id AND ca.is_verified = TRUE) as verified_achievements
		FROM users u
		LEFT JOIN coach_ratings cr ON cr.coach_id = u.id
		` + where + `
		GROUP BY u.id ` + coachSortOrder(sortBy) + `
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Printf("ERROR listing coaches: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coaches")
		return
	}
	defer rows.Close()

	coaches := []models.CoachListItem{}
	for rows.Next() {
		var c models.CoachListItem
		if err := rows.Scan(&c.ID, &c.Name, &c.AvatarURL, &c.CoachDescription,
			&c.CoachLocality, &c.CoachLevel,
			&c.AvgRating, &c.RatingCount, &c.VerifiedCount); err != nil {
			logErr("scan coach list row", err)
			continue
		}
		coaches = append(coaches, c)
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

	var profile models.CoachPublicProfile
	var avatarURL, description sql.NullString
	err = h.DB.QueryRow(`
		SELECT u.id, u.name, u.avatar_url, u.coach_description,
			COALESCE(AVG(cr.rating), 0) as avg_rating,
			COUNT(cr.id) as rating_count
		FROM users u
		LEFT JOIN coach_ratings cr ON cr.coach_id = u.id
		WHERE u.id = ? AND u.is_coach = TRUE
		GROUP BY u.id
	`, coachID).Scan(&profile.ID, &profile.Name, &avatarURL, &description,
		&profile.AvgRating, &profile.RatingCount)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Coach not found")
		return
	}
	if err != nil {
		log.Printf("ERROR fetching coach profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coach profile")
		return
	}
	if avatarURL.Valid {
		profile.AvatarURL = avatarURL.String
	}
	if description.Valid {
		profile.CoachDescription = description.String
	}

	// Fetch achievements
	achRows, err := h.DB.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), is_verified,
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? ORDER BY event_date DESC
	`, coachID)
	if err == nil {
		defer achRows.Close()
		for achRows.Next() {
			var a models.CoachAchievement
			var verifiedAt sql.NullString
			if err := achRows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
				&a.DistanceKm, &a.ResultTime, &a.Position, &a.IsVerified,
				&a.VerifiedBy, &verifiedAt, &a.CreatedAt); err != nil {
				logErr("scan coach achievement row", err)
				continue
			}
			if verifiedAt.Valid {
				a.VerifiedAt = verifiedAt.String
			}
			profile.Achievements = append(profile.Achievements, a)
		}
	}
	if profile.Achievements == nil {
		profile.Achievements = []models.CoachAchievement{}
	}

	// Fetch ratings
	ratRows, err := h.DB.Query(`
		SELECT cr.id, cr.coach_id, cr.student_id, cr.rating, COALESCE(cr.comment, ''),
			u.name as student_name, cr.created_at, cr.updated_at
		FROM coach_ratings cr
		JOIN users u ON u.id = cr.student_id
		WHERE cr.coach_id = ? ORDER BY cr.updated_at DESC
	`, coachID)
	if err == nil {
		defer ratRows.Close()
		for ratRows.Next() {
			var rt models.CoachRating
			if err := ratRows.Scan(&rt.ID, &rt.CoachID, &rt.StudentID, &rt.Rating,
				&rt.Comment, &rt.StudentName, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
				logErr("scan coach rating row", err)
				continue
			}
			profile.Ratings = append(profile.Ratings, rt)
		}
	}
	if profile.Ratings == nil {
		profile.Ratings = []models.CoachRating{}
	}

	writeJSON(w, http.StatusOK, profile)
}

func coachSortOrder(sortBy string) string {
	switch sortBy {
	case "name":
		return "ORDER BY u.name ASC"
	case "newest":
		return "ORDER BY u.created_at DESC"
	case "oldest":
		return "ORDER BY u.created_at ASC"
	default:
		return "ORDER BY avg_rating DESC"
	}
}
