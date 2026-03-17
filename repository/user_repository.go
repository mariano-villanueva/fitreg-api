package repository

import (
	"database/sql"

	"github.com/fitreg/api/models"
)

type userRepository struct {
	db *sql.DB
}

// NewUserRepository constructs a UserRepository backed by MySQL.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByGoogleID(googleID string) (models.UserRow, error) {
	var row models.UserRow
	err := r.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE google_id = ?
	`, googleID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func (r *userRepository) Create(googleID, email, name, avatarURL string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO users (google_id, email, name, avatar_url) VALUES (?, ?, ?, ?)
	`, googleID, email, name, avatarURL)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *userRepository) UpdateOnLogin(googleID, email, name, picture string) error {
	_, err := r.db.Exec(`
		UPDATE users SET email = ?, name = ?, avatar_url = ?, updated_at = NOW() WHERE google_id = ?
	`, email, name, picture, googleID)
	return err
}

func (r *userRepository) GetByID(id int64) (models.UserRow, error) {
	var row models.UserRow
	err := r.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func (r *userRepository) HasActiveCoach(id int64) (bool, error) {
	var hasCoach bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM coach_students WHERE student_id = ? AND status = 'active')", id).Scan(&hasCoach)
	return hasCoach, err
}

func (r *userRepository) GetActiveCoach(studentID int64) (coachID int64, name, avatar string, found bool) {
	var coachAvatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '')
		FROM coach_students cs
		JOIN users u ON u.id = cs.coach_id
		WHERE cs.student_id = ? AND cs.status = 'active'
		LIMIT 1
	`, studentID).Scan(&coachID, &name, &coachAvatar)
	if err != nil {
		return 0, "", "", false
	}
	if coachAvatar.Valid {
		avatar = coachAvatar.String
	}
	return coachID, name, avatar, true
}

func (r *userRepository) UpdateProfile(id int64, req models.UpdateProfileRequest) error {
	var birthDate interface{} = req.BirthDate
	if req.BirthDate == "" {
		birthDate = nil
	}
	var sex interface{} = req.Sex
	if req.Sex == "" {
		sex = nil
	}

	_, err := r.db.Exec(`
		UPDATE users SET name = ?, sex = ?, birth_date = ?, weight_kg = ?, height_cm = ?, language = ?, onboarding_completed = ?, updated_at = NOW() WHERE id = ?
	`, req.Name, sex, birthDate, req.WeightKg, req.HeightCm, req.Language, req.OnboardingCompleted, id)
	return err
}

func (r *userRepository) IsCoach(id int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", id).Scan(&isCoach)
	return isCoach, err
}

func (r *userRepository) HasPendingCoachRequest(id int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, id).Scan(&count)
	return count > 0, err
}

func (r *userRepository) SetCoachLocality(id int64, locality, level string) error {
	_, err := r.db.Exec("UPDATE users SET coach_locality = ?, coach_level = ?, updated_at = NOW() WHERE id = ?",
		locality, level, id)
	return err
}

func (r *userRepository) GetNameAndAvatar(id int64) (name, avatar string, err error) {
	err = r.db.QueryRow("SELECT COALESCE(name, ''), COALESCE(custom_avatar, '') FROM users WHERE id = ?", id).Scan(&name, &avatar)
	return
}

func (r *userRepository) GetAdminIDs() ([]int64, error) {
	rows, err := r.db.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *userRepository) UploadAvatar(id int64, image string) error {
	_, err := r.db.Exec("UPDATE users SET custom_avatar = ?, updated_at = NOW() WHERE id = ?", image, id)
	return err
}

func (r *userRepository) DeleteAvatar(id int64) error {
	_, err := r.db.Exec("UPDATE users SET custom_avatar = NULL, updated_at = NOW() WHERE id = ?", id)
	return err
}
