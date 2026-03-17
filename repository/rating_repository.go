package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type ratingRepository struct {
	db *sql.DB
}

// NewRatingRepository constructs a RatingRepository backed by MySQL.
func NewRatingRepository(db *sql.DB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) IsStudentOf(coachID, studentID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ?", coachID, studentID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *ratingRepository) Upsert(coachID, studentID int64, rating int, comment string) error {
	_, err := r.db.Exec(`
		INSERT INTO coach_ratings (coach_id, student_id, rating, comment)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE rating = VALUES(rating), comment = VALUES(comment), updated_at = NOW()
	`, coachID, studentID, rating, comment)
	return err
}

func (r *ratingRepository) List(coachID int64) ([]models.CoachRating, error) {
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
			log.Printf("ERROR scan rating row: %v", err)
			continue
		}
		ratings = append(ratings, rt)
	}
	return ratings, nil
}
