package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type achievementRepository struct {
	db *sql.DB
}

func NewAchievementRepository(db *sql.DB) AchievementRepository {
	return &achievementRepository{db: db}
}

func (r *achievementRepository) List(coachID int64) ([]models.CoachAchievement, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), COALESCE(extra_info, ''),
			image_file_id, is_public, is_verified, COALESCE(rejection_reason, ''),
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? ORDER BY event_date DESC
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
			log.Printf("WARN scan achievement row: %v", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

func (r *achievementRepository) Create(coachID int64, req models.CreateAchievementRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO coach_achievements (coach_id, event_name, event_date, distance_km, result_time, position, extra_info, image_file_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, coachID, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *achievementRepository) GetForEdit(achID, coachID int64) (bool, string, error) {
	var isVerified bool
	var rejectionReason sql.NullString
	err := r.db.QueryRow(
		"SELECT is_verified, rejection_reason FROM coach_achievements WHERE id = ? AND coach_id = ?",
		achID, coachID,
	).Scan(&isVerified, &rejectionReason)
	if err != nil {
		return false, "", err
	}
	reason := ""
	if rejectionReason.Valid {
		reason = rejectionReason.String
	}
	return isVerified, reason, nil
}

func (r *achievementRepository) Update(achID, coachID int64, req models.UpdateAchievementRequest) error {
	_, err := r.db.Exec(`
		UPDATE coach_achievements SET event_name = ?, event_date = ?, distance_km = ?, result_time = ?,
			position = ?, extra_info = ?, image_file_id = ?, rejection_reason = NULL
		WHERE id = ? AND coach_id = ?
	`, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID, achID, coachID)
	return err
}

func (r *achievementRepository) Delete(achID, coachID int64) (bool, error) {
	result, err := r.db.Exec(
		"DELETE FROM coach_achievements WHERE id = ? AND coach_id = ?",
		achID, coachID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *achievementRepository) SetVisibility(achID, coachID int64, isPublic bool) (bool, error) {
	result, err := r.db.Exec(
		"UPDATE coach_achievements SET is_public = ? WHERE id = ? AND coach_id = ?",
		isPublic, achID, coachID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *achievementRepository) IsCoach(userID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	return isCoach, err
}

func (r *achievementRepository) GetAdminIDs() ([]int64, error) {
	rows, err := r.db.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *achievementRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}
