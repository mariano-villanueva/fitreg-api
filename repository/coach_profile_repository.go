package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type coachProfileRepository struct {
	db *sql.DB
}

// NewCoachProfileRepository constructs a CoachProfileRepository backed by MySQL.
func NewCoachProfileRepository(db *sql.DB) CoachProfileRepository {
	return &coachProfileRepository{db: db}
}

func (r *coachProfileRepository) UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error {
	_, err := r.db.Exec("UPDATE users SET coach_description = ?, coach_public = ?, updated_at = NOW() WHERE id = ?",
		req.CoachDescription, req.CoachPublic, coachID)
	return err
}

func (r *coachProfileRepository) IsCoach(userID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	return isCoach, err
}

func (r *coachProfileRepository) ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
	where := "WHERE u.is_coach = TRUE AND u.coach_public = TRUE"
	args := []interface{}{}

	if search != "" {
		where += " AND (u.name LIKE ? OR u.coach_description LIKE ? OR u.coach_locality LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
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
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("ERROR count coaches: %v", err)
	}

	query := `
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '') as avatar_url,
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	coaches := []models.CoachListItem{}
	for rows.Next() {
		var c models.CoachListItem
		if err := rows.Scan(&c.ID, &c.Name, &c.AvatarURL, &c.CoachDescription,
			&c.CoachLocality, &c.CoachLevel,
			&c.AvgRating, &c.RatingCount, &c.VerifiedCount); err != nil {
			log.Printf("ERROR scan coach list row: %v", err)
			continue
		}
		coaches = append(coaches, c)
	}

	return coaches, total, nil
}

func (r *coachProfileRepository) GetCoachProfile(coachID int64) (models.CoachPublicProfile, error) {
	var profile models.CoachPublicProfile
	var avatarURL, description sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.name, u.custom_avatar, u.coach_description,
			COALESCE(AVG(cr.rating), 0) as avg_rating,
			COUNT(cr.id) as rating_count
		FROM users u
		LEFT JOIN coach_ratings cr ON cr.coach_id = u.id
		WHERE u.id = ? AND u.is_coach = TRUE
		GROUP BY u.id
	`, coachID).Scan(&profile.ID, &profile.Name, &avatarURL, &description,
		&profile.AvgRating, &profile.RatingCount)
	if err != nil {
		return profile, err
	}
	if avatarURL.Valid && avatarURL.String != "" {
		profile.AvatarURL = avatarURL.String
	}
	if description.Valid {
		profile.CoachDescription = description.String
	}
	return profile, nil
}

func (r *coachProfileRepository) IsStudentOf(coachID, studentID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ? AND status = 'active'", coachID, studentID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *coachProfileRepository) CountStudents(coachID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM coach_students WHERE coach_id = ? AND status = 'active'", coachID).Scan(&count)
	return count, err
}

func (r *coachProfileRepository) CountVerifiedAchievements(coachID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM coach_achievements WHERE coach_id = ? AND is_verified = TRUE", coachID).Scan(&count)
	return count, err
}

func (r *coachProfileRepository) GetAchievements(coachID int64) ([]models.CoachAchievement, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), COALESCE(extra_info, ''),
			image_file_id, is_public, is_verified, COALESCE(rejection_reason, ''),
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? AND is_public = TRUE ORDER BY event_date DESC
	`, coachID)
	if err != nil {
		return nil, err
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
			log.Printf("ERROR scan coach achievement row: %v", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
		// truncate event date
		if len(a.EventDate) >= 10 {
			a.EventDate = a.EventDate[:10]
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

func (r *coachProfileRepository) GetRatings(coachID int64) ([]models.CoachRating, error) {
	rows, err := r.db.Query(`
		SELECT cr.id, cr.coach_id, cr.student_id, cr.rating, COALESCE(cr.comment, ''),
			u.name as student_name, cr.created_at, cr.updated_at
		FROM coach_ratings cr
		JOIN users u ON u.id = cr.student_id
		WHERE cr.coach_id = ? ORDER BY cr.updated_at DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.CoachRating{}
	for rows.Next() {
		var rt models.CoachRating
		if err := rows.Scan(&rt.ID, &rt.CoachID, &rt.StudentID, &rt.Rating,
			&rt.Comment, &rt.StudentName, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			log.Printf("ERROR scan coach rating row: %v", err)
			continue
		}
		ratings = append(ratings, rt)
	}
	return ratings, nil
}

func (r *coachProfileRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}

// coachSortOrder returns the ORDER BY clause for coach listing.
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
