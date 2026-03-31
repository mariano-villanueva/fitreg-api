package repository

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/fitreg/api/models"
)

// ErrStatusConflict is returned when a workout cannot be transitioned
// because it is already in a terminal state (completed or skipped).
var ErrStatusConflict = errors.New("workout already finalized")

type workoutRepository struct {
	db *sql.DB
}

func NewWorkoutRepository(db *sql.DB) WorkoutRepository {
	return &workoutRepository{db: db}
}

const workoutSelectCols = `id, user_id, coach_id, title, description, type, notes, due_date,
	distance_km, duration_seconds, expected_fields,
	result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
	avg_pace, calories, image_file_id, status, created_at, updated_at`

func scanWorkoutRow(row interface{ Scan(...interface{}) error }) (models.Workout, error) {
	var wo models.Workout
	var coachID sql.NullInt64
	var title, description, workoutType, notes, avgPace sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds, calories sql.NullInt64
	var expectedFields sql.NullString
	var resultDistKm sql.NullFloat64
	var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
	var dueDate sql.NullString

	err := row.Scan(
		&wo.ID, &wo.UserID, &coachID,
		&title, &description, &workoutType, &notes, &dueDate,
		&distanceKm, &durationSeconds,
		&expectedFields,
		&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
		&avgPace, &calories, &imageFileID,
		&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
	)
	if err != nil {
		return wo, err
	}
	if coachID.Valid {
		v := coachID.Int64
		wo.CoachID = &v
	}
	if title.Valid {
		wo.Title = title.String
	}
	if description.Valid {
		wo.Description = description.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
	}
	if dueDate.Valid {
		wo.DueDate = truncateDate(dueDate.String)
	}
	if distanceKm.Valid {
		wo.DistanceKm = distanceKm.Float64
	}
	if durationSeconds.Valid {
		wo.DurationSeconds = int(durationSeconds.Int64)
	}
	if expectedFields.Valid {
		wo.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	if resultDistKm.Valid {
		v := resultDistKm.Float64
		wo.ResultDistanceKm = &v
	}
	if resultTimeSec.Valid {
		v := int(resultTimeSec.Int64)
		wo.ResultTimeSeconds = &v
	}
	if resultHR.Valid {
		v := int(resultHR.Int64)
		wo.ResultHeartRate = &v
	}
	if resultFeeling.Valid {
		v := int(resultFeeling.Int64)
		wo.ResultFeeling = &v
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if calories.Valid {
		wo.Calories = int(calories.Int64)
	}
	if imageFileID.Valid {
		v := imageFileID.Int64
		wo.ImageFileID = &v
	}
	return wo, nil
}

func scanWorkoutRows(rows *sql.Rows) ([]models.Workout, error) {
	workouts := []models.Workout{}
	for rows.Next() {
		wo, err := scanWorkoutRow(rows)
		if err != nil {
			return nil, err
		}
		workouts = append(workouts, wo)
	}
	return workouts, rows.Err()
}

func (r *workoutRepository) List(userID int64, startDate, endDate string) ([]models.Workout, error) {
	query := `SELECT ` + workoutSelectCols + ` FROM workouts WHERE user_id = ?`
	args := []interface{}{userID}
	if startDate != "" {
		query += ` AND due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND due_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY due_date DESC`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkoutRows(rows)
}

func (r *workoutRepository) GetByID(id int64) (models.Workout, error) {
	row := r.db.QueryRow(`SELECT `+workoutSelectCols+` FROM workouts WHERE id = ?`, id)
	return scanWorkoutRow(row)
}

func (r *workoutRepository) Create(userID int64, req models.CreateWorkoutRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO workouts
		  (user_id, coach_id, due_date, distance_km, duration_seconds, avg_pace, calories,
		   result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
		   type, notes, status)
		VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed')
	`, userID, req.DueDate, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories,
		req.ResultDistanceKm, req.ResultTimeSeconds, req.ResultHeartRate, req.ResultFeeling,
		req.Type, req.Notes)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *workoutRepository) Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE workouts SET
		  due_date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?,
		  result_distance_km = ?, result_time_seconds = ?, result_heart_rate = ?, result_feeling = ?,
		  type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ? AND coach_id IS NULL
	`, req.DueDate, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories,
		req.ResultDistanceKm, req.ResultTimeSeconds, req.ResultHeartRate, req.ResultFeeling,
		req.Type, req.Notes, id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (r *workoutRepository) Delete(id, userID int64) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ? AND coach_id IS NULL`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (r *workoutRepository) CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error) {
	var ef interface{}
	if len(req.ExpectedFields) > 0 {
		b, err := json.Marshal(req.ExpectedFields)
		if err != nil {
			return models.Workout{}, err
		}
		ef = string(b)
	}
	result, err := r.db.Exec(`
		INSERT INTO workouts
		  (user_id, coach_id, title, description, type, distance_km, duration_seconds, notes,
		   expected_fields, due_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')
	`, req.StudentID, coachID, req.Title, req.Description, req.Type,
		req.DistanceKm, req.DurationSeconds, req.Notes, ef, req.DueDate)
	if err != nil {
		return models.Workout{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Workout{}, err
	}
	return r.GetByID(id)
}

func (r *workoutRepository) ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error) {
	where := `WHERE w.coach_id = ?`
	args := []interface{}{coachID}
	if studentID != nil {
		where += ` AND w.user_id = ?`
		args = append(args, *studentID)
	}
	if statusFilter != "" {
		where += ` AND w.status = ?`
		args = append(args, statusFilter)
	}
	if startDate != "" {
		where += ` AND w.due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		where += ` AND w.due_date <= ?`
		args = append(args, endDate)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM workouts w `+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := make([]interface{}, len(args))
	copy(listArgs, args)

	query := `SELECT w.id, w.user_id, w.coach_id, w.title, w.description, w.type, w.notes, w.due_date,
		w.distance_km, w.duration_seconds, w.expected_fields,
		w.result_distance_km, w.result_time_seconds, w.result_heart_rate, w.result_feeling,
		w.avg_pace, w.calories, w.image_file_id, w.status, w.created_at, w.updated_at,
		u.name
		FROM workouts w JOIN users u ON u.id = w.user_id ` + where + ` ORDER BY w.due_date DESC`
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		listArgs = append(listArgs, limit, offset)
	}
	rows, err := r.db.Query(query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var cID sql.NullInt64
		var title, description, workoutType, notes, avgPace, userName sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds, calories sql.NullInt64
		var expectedFields sql.NullString
		var resultDistKm sql.NullFloat64
		var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
		var dueDate sql.NullString
		if err := rows.Scan(
			&wo.ID, &wo.UserID, &cID,
			&title, &description, &workoutType, &notes, &dueDate,
			&distanceKm, &durationSeconds,
			&expectedFields,
			&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
			&avgPace, &calories, &imageFileID,
			&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
			&userName,
		); err != nil {
			return nil, 0, err
		}
		if cID.Valid {
			v := cID.Int64
			wo.CoachID = &v
		}
		if title.Valid {
			wo.Title = title.String
		}
		if description.Valid {
			wo.Description = description.String
		}
		if workoutType.Valid {
			wo.Type = workoutType.String
		}
		if notes.Valid {
			wo.Notes = notes.String
		}
		if dueDate.Valid {
			wo.DueDate = truncateDate(dueDate.String)
		}
		if distanceKm.Valid {
			wo.DistanceKm = distanceKm.Float64
		}
		if durationSeconds.Valid {
			wo.DurationSeconds = int(durationSeconds.Int64)
		}
		if expectedFields.Valid {
			wo.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		if resultDistKm.Valid {
			v := resultDistKm.Float64
			wo.ResultDistanceKm = &v
		}
		if resultTimeSec.Valid {
			v := int(resultTimeSec.Int64)
			wo.ResultTimeSeconds = &v
		}
		if resultHR.Valid {
			v := int(resultHR.Int64)
			wo.ResultHeartRate = &v
		}
		if resultFeeling.Valid {
			v := int(resultFeeling.Int64)
			wo.ResultFeeling = &v
		}
		if avgPace.Valid {
			wo.AvgPace = avgPace.String
		}
		if calories.Valid {
			wo.Calories = int(calories.Int64)
		}
		if imageFileID.Valid {
			v := imageFileID.Int64
			wo.ImageFileID = &v
		}
		if userName.Valid {
			wo.UserName = userName.String
		}
		workouts = append(workouts, wo)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return workouts, total, nil
}

func (r *workoutRepository) GetCoachWorkout(workoutID, coachID int64) (models.Workout, error) {
	row := r.db.QueryRow(`SELECT `+workoutSelectCols+` FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID)
	return scanWorkoutRow(row)
}

func (r *workoutRepository) UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error) {
	var ef interface{}
	if len(req.ExpectedFields) > 0 {
		b, err := json.Marshal(req.ExpectedFields)
		if err != nil {
			return models.Workout{}, err
		}
		ef = string(b)
	}
	res, err := r.db.Exec(`
		UPDATE workouts SET
		  title = ?, description = ?, type = ?, distance_km = ?, duration_seconds = ?,
		  notes = ?, expected_fields = ?, due_date = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ? AND status = 'pending'
	`, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds,
		req.Notes, ef, req.DueDate, workoutID, coachID)
	if err != nil {
		return models.Workout{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return models.Workout{}, err
	}
	if n == 0 {
		return models.Workout{}, sql.ErrNoRows
	}
	return r.GetByID(workoutID)
}

func (r *workoutRepository) GetWorkoutStatus(workoutID, coachID int64) (string, error) {
	var status string
	err := r.db.QueryRow(`SELECT status FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID).Scan(&status)
	return status, err
}

func (r *workoutRepository) DeleteCoachWorkout(workoutID, coachID int64) error {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *workoutRepository) GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error) {
	query := `SELECT ` + workoutSelectCols + ` FROM workouts WHERE user_id = ?`
	args := []interface{}{studentID}
	if startDate != "" {
		query += ` AND due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND due_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY due_date ASC`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkoutRows(rows)
}

func (r *workoutRepository) UpdateStatus(workoutID, studentID int64, req models.UpdateWorkoutStatusRequest) (int64, string, error) {
	res, err := r.db.Exec(`
		UPDATE workouts SET
		  status = ?, result_time_seconds = ?, result_distance_km = ?,
		  result_heart_rate = ?, result_feeling = ?, image_file_id = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ? AND coach_id IS NOT NULL AND status = 'pending'
	`, req.Status, req.ResultTimeSeconds, req.ResultDistanceKm,
		req.ResultHeartRate, req.ResultFeeling, req.ImageFileID, workoutID, studentID)
	if err != nil {
		return 0, "", err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, "", err
	}
	if n == 0 {
		// Distinguish: not found vs already finalized
		var currentStatus string
		err := r.db.QueryRow(
			`SELECT status FROM workouts WHERE id = ? AND user_id = ? AND coach_id IS NOT NULL`,
			workoutID, studentID,
		).Scan(&currentStatus)
		if err == sql.ErrNoRows {
			return 0, "", sql.ErrNoRows
		}
		if err != nil {
			return 0, "", err
		}
		return 0, "", ErrStatusConflict
	}
	// Fetch coachID and title for notification
	var coachID int64
	var title string
	err = r.db.QueryRow(
		`SELECT coach_id, COALESCE(title,'') FROM workouts WHERE id = ?`,
		workoutID,
	).Scan(&coachID, &title)
	if err != nil {
		return 0, "", err
	}
	return coachID, title, nil
}

func (r *workoutRepository) GetSegments(workoutID int64) ([]models.WorkoutSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, parent_id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
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
		var parentID sql.NullInt64
		if err := rows.Scan(&s.ID, &parentID, &s.WorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			s.ParentID = &v
		}
		segments = append(segments, s)
	}
	return segments, rows.Err()
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
			INSERT INTO workout_segments
			  (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
			   work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *workoutRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}
