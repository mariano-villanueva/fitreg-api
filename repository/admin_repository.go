package repository

import (
	"database/sql"
	"fmt"

	"github.com/fitreg/api/models"
)

type adminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) IsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

func (r *adminRepository) GetStats() (totalUsers, totalCoaches, totalRatings, pendingAchievements int, err error) {
	if e := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers); e != nil {
		err = e
	}
	if e := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_coach = TRUE").Scan(&totalCoaches); e != nil {
		err = e
	}
	if e := r.db.QueryRow("SELECT COUNT(*) FROM coach_ratings").Scan(&totalRatings); e != nil {
		err = e
	}
	if e := r.db.QueryRow("SELECT COUNT(*) FROM coach_achievements WHERE is_verified = FALSE AND rejection_reason IS NULL").Scan(&pendingAchievements); e != nil {
		err = e
	}
	return
}

func (r *adminRepository) ListUsers(search, role, sortCol, sortOrder string, limit, offset int) (users []models.AdminUser, total int, err error) {
	// Whitelist sort column
	allowedSort := map[string]bool{
		"id":         true,
		"name":       true,
		"email":      true,
		"created_at": true,
	}
	if !allowedSort[sortCol] {
		sortCol = "id"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
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
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if e := r.db.QueryRow("SELECT COUNT(*) FROM users "+where, countArgs...).Scan(&total); e != nil {
		err = e
		return
	}

	// Paginated SELECT
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT id, email, COALESCE(name, '') as name,
			COALESCE(custom_avatar, avatar_url, '') as avatar_url,
			COALESCE(is_coach, FALSE), COALESCE(is_admin, FALSE), created_at
		FROM users %s ORDER BY %s %s LIMIT ? OFFSET ?`, where, sortCol, sortOrder)

	rows, e := r.db.Query(query, args...)
	if e != nil {
		err = e
		return
	}
	defer rows.Close()

	users = []models.AdminUser{}
	for rows.Next() {
		var u models.AdminUser
		if e := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL,
			&u.IsCoach, &u.IsAdmin, &u.CreatedAt); e != nil {
			continue
		}
		users = append(users, u)
	}
	return
}

func (r *adminRepository) UpdateUserRoles(targetID int64, isCoach, isAdmin *bool) error {
	if isCoach == nil && isAdmin == nil {
		return nil
	}
	if isCoach != nil {
		if _, err := r.db.Exec("UPDATE users SET is_coach = ?, updated_at = NOW() WHERE id = ?", *isCoach, targetID); err != nil {
			return err
		}
	}
	if isAdmin != nil {
		if _, err := r.db.Exec("UPDATE users SET is_admin = ?, updated_at = NOW() WHERE id = ?", *isAdmin, targetID); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminRepository) ListPendingAchievements() ([]models.AdminPendingAchievement, error) {
	rows, err := r.db.Query(`
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
		return nil, err
	}
	defer rows.Close()

	achievements := []models.AdminPendingAchievement{}
	for rows.Next() {
		var a models.AdminPendingAchievement
		if err := rows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.ExtraInfo,
			&a.ImageFileID, &a.CreatedAt, &a.CoachName); err != nil {
			continue
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

func (r *adminRepository) VerifyAchievement(achID, adminID int64) (coachID int64, eventName string, err error) {
	result, e := r.db.Exec(`
		UPDATE coach_achievements SET is_verified = TRUE, rejection_reason = NULL, verified_by = ?, verified_at = NOW()
		WHERE id = ? AND is_verified = FALSE
	`, adminID, achID)
	if e != nil {
		err = e
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		err = sql.ErrNoRows
		return
	}
	err = r.db.QueryRow("SELECT coach_id, event_name FROM coach_achievements WHERE id = ?", achID).Scan(&coachID, &eventName)
	return
}

func (r *adminRepository) RejectAchievement(achID int64, reason string) (coachID int64, eventName string, err error) {
	// Fetch achievement info before rejecting
	if e := r.db.QueryRow("SELECT coach_id, event_name FROM coach_achievements WHERE id = ? AND is_verified = FALSE AND rejection_reason IS NULL", achID).Scan(&coachID, &eventName); e != nil {
		err = e
		return
	}

	result, e := r.db.Exec("UPDATE coach_achievements SET rejection_reason = ? WHERE id = ? AND is_verified = FALSE", reason, achID)
	if e != nil {
		err = e
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		err = sql.ErrNoRows
		return
	}
	return
}

func (r *adminRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}
