package repository

import (
	"database/sql"

	"github.com/fitreg/api/models"
)

type workoutRepository struct {
	db *sql.DB
}

func NewWorkoutRepository(db *sql.DB) WorkoutRepository {
	return &workoutRepository{db: db}
}

func (r *workoutRepository) List(userID int64) ([]models.Workout, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var avgPace, workoutType, notes sql.NullString
		if err := rows.Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
			&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
			return nil, err
		}
		if avgPace.Valid {
			wo.AvgPace = avgPace.String
		}
		if workoutType.Valid {
			wo.Type = workoutType.String
		}
		if notes.Valid {
			wo.Notes = notes.String
		}
		workouts = append(workouts, wo)
	}
	return workouts, nil
}

func (r *workoutRepository) GetByID(id int64) (models.Workout, error) {
	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	err := r.db.QueryRow(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts WHERE id = ?
	`, id).Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt)
	if err != nil {
		return wo, err
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
	}
	return wo, nil
}

func (r *workoutRepository) ExistsByOwner(id, userID int64) bool {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM workouts WHERE id = ? AND user_id = ?", id, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	return err == nil
}

func (r *workoutRepository) Create(userID int64, req models.CreateWorkoutRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *workoutRepository) Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE workouts SET date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?, avg_heart_rate = ?, feeling = ?, type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ?
	`, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes, id, userID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *workoutRepository) Delete(id, userID int64) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *workoutRepository) GetSegments(workoutID int64) ([]models.WorkoutSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
			COALESCE(value, 0), COALESCE(unit, ''), COALESCE(intensity, ''),
			COALESCE(work_value, 0), COALESCE(work_unit, ''), COALESCE(work_intensity, ''),
			COALESCE(rest_value, 0), COALESCE(rest_unit, ''), COALESCE(rest_intensity, '')
		FROM workout_segments WHERE workout_id = ? ORDER BY order_index
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, nil
}

func (r *workoutRepository) ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM workout_segments WHERE workout_id = ?", workoutID); err != nil {
		return err
	}
	for i, seg := range segs {
		if _, err := tx.Exec(`
			INSERT INTO workout_segments (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return tx.Commit()
}
