package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/fitreg/api/models"
)

// ErrStatusConflict is returned when an assigned workout cannot be transitioned
// because it is already in a terminal state (completed or skipped).
var ErrStatusConflict = errors.New("workout already finalized")

type coachRepository struct {
	db *sql.DB
}

func NewCoachRepository(db *sql.DB) CoachRepository {
	return &coachRepository{db: db}
}

func (r *coachRepository) IsCoach(userID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	return isCoach, err
}

func (r *coachRepository) IsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return isAdmin, err
}

func (r *coachRepository) IsStudentOf(coachID, studentID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ? AND status = 'active'", coachID, studentID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *coachRepository) GetStudents(coachID int64) ([]models.CoachStudentInfo, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.name, u.email, COALESCE(u.custom_avatar, '') as avatar_url
		FROM users u
		JOIN coach_students cs ON u.id = cs.student_id
		WHERE cs.coach_id = ? AND cs.status = 'active'
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []models.CoachStudentInfo{}
	for rows.Next() {
		var s models.CoachStudentInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.AvatarURL); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *coachRepository) GetRelationship(csID int64) (coachID, studentID int64, status string, err error) {
	err = r.db.QueryRow("SELECT coach_id, student_id, status FROM coach_students WHERE id = ?", csID).Scan(&coachID, &studentID, &status)
	return
}

func (r *coachRepository) EndRelationship(csID int64) error {
	_, err := r.db.Exec("UPDATE coach_students SET status = 'finished', finished_at = NOW() WHERE id = ?", csID)
	return err
}

func (r *coachRepository) GetStudentWorkouts(studentID int64) ([]models.Workout, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY date DESC
	`, studentID)
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

func (r *coachRepository) ListAssignedWorkouts(coachID int64, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
	query := `
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name,
			(SELECT COUNT(*) FROM assignment_messages am WHERE am.assigned_workout_id = aw.id AND am.sender_id != ? AND am.is_read = FALSE)
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.coach_id = ?
	`
	args := []interface{}{coachID, coachID}

	if studentID != 0 {
		query += " AND aw.student_id = ?"
		args = append(args, studentID)
	}

	if statusFilter == "pending" {
		query += " AND aw.status = 'pending'"
	} else if statusFilter == "finished" {
		query += " AND aw.status IN ('completed', 'skipped')"
	}

	if startDate != "" && endDate != "" {
		query += " AND aw.due_date >= ? AND aw.due_date <= ?"
		args = append(args, startDate, endDate)
	}

	// Count total before pagination
	var total int
	countQuery := "SELECT COUNT(*) FROM assigned_workouts aw WHERE aw.coach_id = ?"
	countArgs := []interface{}{coachID}
	if studentID != 0 {
		countQuery += " AND aw.student_id = ?"
		countArgs = append(countArgs, studentID)
	}
	if statusFilter == "pending" {
		countQuery += " AND aw.status = 'pending'"
	} else if statusFilter == "finished" {
		countQuery += " AND aw.status IN ('completed', 'skipped')"
	}
	if startDate != "" && endDate != "" {
		countQuery += " AND aw.due_date >= ? AND aw.due_date <= ?"
		countArgs = append(countArgs, startDate, endDate)
	}
	if err := r.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		log.Printf("count assigned workouts: %v", err)
	}

	query += " ORDER BY aw.due_date DESC"
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	workouts := []models.AssignedWorkout{}
	for rows.Next() {
		var aw models.AssignedWorkout
		var description, notes, dueDate, expectedFields, workoutType sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds sql.NullInt64
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &workoutType,
			&distanceKm, &durationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
			&aw.CreatedAt, &aw.UpdatedAt, &aw.StudentName, &aw.UnreadMessageCount); err != nil {
			return nil, 0, err
		}
		if description.Valid {
			aw.Description = description.String
		}
		if notes.Valid {
			aw.Notes = notes.String
		}
		if dueDate.Valid {
			aw.DueDate = truncateDate(dueDate.String)
		}
		if expectedFields.Valid {
			aw.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		if workoutType.Valid {
			aw.Type = workoutType.String
		}
		if distanceKm.Valid {
			aw.DistanceKm = distanceKm.Float64
		}
		if durationSeconds.Valid {
			aw.DurationSeconds = int(durationSeconds.Int64)
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = r.FetchSegments(workouts[i].ID)
		if workouts[i].ImageFileID != nil {
			if uuid, err := r.GetFileUUID(*workouts[i].ImageFileID); err == nil {
				workouts[i].ImageURL = "/api/files/" + uuid + "/download"
			}
		}
	}

	return workouts, total, nil
}

func (r *coachRepository) CreateAssignedWorkout(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
	var dueDateVal interface{}
	if req.DueDate != "" {
		dueDateVal = req.DueDate
	}

	if len(req.ExpectedFields) == 0 {
		req.ExpectedFields = []string{"feeling"}
	}
	expectedFieldsJSON, err := json.Marshal(req.ExpectedFields)
	if err != nil {
		log.Printf("marshal expected fields: %v", err)
	}

	log.Printf("Creating assigned workout: coach=%d student=%d title=%s type=%s due=%v", coachID, req.StudentID, req.Title, req.Type, dueDateVal)
	result, err := r.db.Exec(`
		INSERT INTO assigned_workouts (coach_id, student_id, title, description, type, distance_km, duration_seconds, notes, expected_fields, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, coachID, req.StudentID, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, expectedFieldsJSON, dueDateVal)
	if err != nil {
		return models.AssignedWorkout{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("get last insert id for assigned workout: %v", err)
	}

	// Insert segments
	for i, seg := range req.Segments {
		log.Printf("Inserting segment %d: type=%s unit=%s intensity=%s work_unit=%s work_intensity=%s rest_unit=%s rest_intensity=%s",
			i, seg.SegmentType, seg.Unit, seg.Intensity, seg.WorkUnit, seg.WorkIntensity, seg.RestUnit, seg.RestIntensity)
		if _, err := r.db.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			log.Printf("ERROR inserting segment %d: %v", i, err)
			return models.AssignedWorkout{}, err
		}
	}

	return r.GetAssignedWorkout(id, coachID)
}

func (r *coachRepository) GetAssignedWorkout(awID, coachID int64) (models.AssignedWorkout, error) {
	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields, workoutType sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds sql.NullInt64
	var studentName string
	err := r.db.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ? AND aw.coach_id = ?
	`, awID, coachID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &workoutType,
		&distanceKm, &durationSeconds, &notes, &expectedFields,
		&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
		&aw.ImageFileID, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt, &studentName)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if workoutType.Valid {
		aw.Type = workoutType.String
	}
	if distanceKm.Valid {
		aw.DistanceKm = distanceKm.Float64
	}
	if durationSeconds.Valid {
		aw.DurationSeconds = int(durationSeconds.Int64)
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.Segments = r.FetchSegments(aw.ID)
	if aw.ImageFileID != nil {
		if uuid, err := r.GetFileUUID(*aw.ImageFileID); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return aw, nil
}

func (r *coachRepository) GetAssignedWorkoutStatus(awID, coachID int64) (string, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM assigned_workouts WHERE id = ? AND coach_id = ?", awID, coachID).Scan(&status)
	return status, err
}

func (r *coachRepository) UpdateAssignedWorkout(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {

	var dueDateVal interface{}
	if req.DueDate != "" {
		dueDateVal = req.DueDate
	}

	if len(req.ExpectedFields) == 0 {
		req.ExpectedFields = []string{"feeling"}
	}
	efJSON, err := json.Marshal(req.ExpectedFields)
	if err != nil {
		log.Printf("marshal expected fields for update: %v", err)
	}

	result, err := r.db.Exec(`
		UPDATE assigned_workouts
		SET title = ?, description = ?, type = ?, distance_km = ?, duration_seconds = ?, notes = ?, expected_fields = ?, due_date = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, efJSON, dueDateVal, awID, coachID)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return models.AssignedWorkout{}, sql.ErrNoRows
	}

	// Replace segments: delete old, insert new
	if _, err := r.db.Exec("DELETE FROM assigned_workout_segments WHERE assigned_workout_id = ?", awID); err != nil {
		log.Printf("delete old segments: %v", err)
	}
	for i, seg := range req.Segments {
		if _, err := r.db.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, awID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			log.Printf("insert updated segment: %v", err)
		}
	}

	// Return updated workout (fetch by ID without coach_id filter for the SELECT back)
	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields, workoutType2 sql.NullString
	var distanceKm2 sql.NullFloat64
	var durationSeconds2 sql.NullInt64
	var studentName string
	err = r.db.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ?
	`, awID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &workoutType2,
		&distanceKm2, &durationSeconds2, &notes, &expectedFields,
		&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
		&aw.ImageFileID, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt, &studentName)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if workoutType2.Valid {
		aw.Type = workoutType2.String
	}
	if distanceKm2.Valid {
		aw.DistanceKm = distanceKm2.Float64
	}
	if durationSeconds2.Valid {
		aw.DurationSeconds = int(durationSeconds2.Int64)
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.Segments = r.FetchSegments(aw.ID)
	if aw.ImageFileID != nil {
		if uuid, err := r.GetFileUUID(*aw.ImageFileID); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return aw, nil
}

func (r *coachRepository) DeleteAssignedWorkout(awID, coachID int64) error {
	result, err := r.db.Exec("DELETE FROM assigned_workouts WHERE id = ? AND coach_id = ?", awID, coachID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("get rows affected for delete assigned workout: %v", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *coachRepository) GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
	query := `
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as coach_name,
			(SELECT COUNT(*) FROM assignment_messages am WHERE am.assigned_workout_id = aw.id AND am.sender_id != ? AND am.is_read = FALSE)
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.coach_id
		WHERE aw.student_id = ?
	`
	args := []interface{}{studentID, studentID}

	if startDate != "" && endDate != "" {
		query += " AND aw.due_date >= ? AND aw.due_date <= ?"
		args = append(args, startDate, endDate)
	}

	query += " ORDER BY aw.due_date ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.AssignedWorkout{}
	for rows.Next() {
		var aw models.AssignedWorkout
		var description, notes, dueDate, expectedFields, workoutType sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds sql.NullInt64
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &workoutType,
			&distanceKm, &durationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
			&aw.CreatedAt, &aw.UpdatedAt, &aw.CoachName, &aw.UnreadMessageCount); err != nil {
			return nil, err
		}
		if description.Valid {
			aw.Description = description.String
		}
		if notes.Valid {
			aw.Notes = notes.String
		}
		if dueDate.Valid {
			aw.DueDate = truncateDate(dueDate.String)
		}
		if expectedFields.Valid {
			aw.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		if workoutType.Valid {
			aw.Type = workoutType.String
		}
		if distanceKm.Valid {
			aw.DistanceKm = distanceKm.Float64
		}
		if durationSeconds.Valid {
			aw.DurationSeconds = int(durationSeconds.Int64)
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = r.FetchSegments(workouts[i].ID)
		if workouts[i].ImageFileID != nil {
			if uuid, err := r.GetFileUUID(*workouts[i].ImageFileID); err == nil {
				workouts[i].ImageURL = "/api/files/" + uuid + "/download"
			}
		}
	}

	return workouts, nil
}

func (r *coachRepository) UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) (coachID int64, workoutTitle string, err error) {
	// Only allow transitioning from 'pending' — this enforces the state machine
	// atomically and prevents race conditions with concurrent completion requests.
	result, err := r.db.Exec(`
		UPDATE assigned_workouts SET status = ?,
			result_time_seconds = ?, result_distance_km = ?, result_heart_rate = ?, result_feeling = ?,
			image_file_id = ?, updated_at = NOW()
		WHERE id = ? AND student_id = ? AND status = 'pending'
	`, req.Status, req.ResultTimeSeconds, req.ResultDistanceKm, req.ResultHeartRate, req.ResultFeeling, req.ImageFileID, awID, studentID)
	if err != nil {
		return 0, "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("get rows affected for update status: %v", err)
	}
	if rowsAffected == 0 {
		// Distinguish: does the record exist at all, or is it already in a terminal state?
		var exists int
		checkErr := r.db.QueryRow(
			"SELECT 1 FROM assigned_workouts WHERE id = ? AND student_id = ?", awID, studentID,
		).Scan(&exists)
		if checkErr == sql.ErrNoRows {
			return 0, "", sql.ErrNoRows
		}
		return 0, "", ErrStatusConflict
	}

	// If completed, create a workout record for the athlete
	if req.Status == "completed" {
		var aw struct {
			Type            string
			DistanceKm      float64
			DurationSeconds int
			Notes           sql.NullString
			DueDate         sql.NullString
		}
		if err := r.db.QueryRow(`SELECT COALESCE(type,'easy'), COALESCE(distance_km,0), COALESCE(duration_seconds,0), notes, due_date
			FROM assigned_workouts WHERE id = ?`, awID).Scan(&aw.Type, &aw.DistanceKm, &aw.DurationSeconds, &aw.Notes, &aw.DueDate); err != nil {
			log.Printf("fetch assigned workout for completion: %v", err)
		}

		finalDistance := aw.DistanceKm
		if req.ResultDistanceKm != nil {
			finalDistance = *req.ResultDistanceKm
		}
		finalDuration := aw.DurationSeconds
		if req.ResultTimeSeconds != nil {
			finalDuration = *req.ResultTimeSeconds
		}
		var avgHR int
		if req.ResultHeartRate != nil {
			avgHR = *req.ResultHeartRate
		}

		var notes interface{}
		if aw.Notes.Valid {
			notes = aw.Notes.String
		}

		var insertErr error
		if aw.DueDate.Valid {
			_, insertErr = r.db.Exec(`INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_heart_rate, type, notes, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
				studentID, truncateDate(aw.DueDate.String), finalDistance, finalDuration, avgHR, aw.Type, notes)
		} else {
			_, insertErr = r.db.Exec(`INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_heart_rate, type, notes, created_at, updated_at)
				VALUES (?, CURDATE(), ?, ?, ?, ?, ?, NOW(), NOW())`,
				studentID, finalDistance, finalDuration, avgHR, aw.Type, notes)
		}
		if insertErr != nil {
			log.Printf("insert workout from completed assignment: %v", insertErr)
		}
	}

	// Fetch coachID and workoutTitle for notification
	if err := r.db.QueryRow("SELECT coach_id, title FROM assigned_workouts WHERE id = ?", awID).Scan(&coachID, &workoutTitle); err != nil {
		log.Printf("fetch coach id and title for status notification: %v", err)
	}

	return coachID, workoutTitle, nil
}

func (r *coachRepository) GetDailySummary(coachID int64, date string) ([]models.DailySummaryItem, error) {
	rows, err := r.db.Query(`
		SELECT
			u.id, u.name,
			CASE WHEN u.custom_avatar IS NOT NULL AND u.custom_avatar != '' THEN u.custom_avatar ELSE u.avatar_url END as avatar,
			aw.id, aw.title, COALESCE(aw.type, ''), aw.distance_km, aw.duration_seconds,
			COALESCE(aw.description, ''), COALESCE(aw.notes, ''), aw.status,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.due_date, aw.created_at
		FROM coach_students cs
		JOIN users u ON u.id = cs.student_id
		LEFT JOIN assigned_workouts aw
			ON aw.student_id = cs.student_id
			AND aw.coach_id = ?
			AND aw.due_date = ?
		WHERE cs.coach_id = ? AND cs.status = 'active'
		ORDER BY u.name ASC, aw.created_at DESC
	`, coachID, date, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[int64]bool{}
	items := []models.DailySummaryItem{}

	for rows.Next() {
		var studentID int64
		var studentName string
		var studentAvatar sql.NullString

		var awID sql.NullInt64
		var awTitle, awType, awDesc, awNotes, awStatus sql.NullString
		var awDist sql.NullFloat64
		var awDur sql.NullInt64
		var awResultTimeSec, awResultHR, awResultFeeling sql.NullInt64
		var awResultDistKm sql.NullFloat64
		var awDueDate sql.NullString
		var awCreatedAt sql.NullTime

		if err := rows.Scan(
			&studentID, &studentName, &studentAvatar,
			&awID, &awTitle, &awType, &awDist, &awDur,
			&awDesc, &awNotes, &awStatus,
			&awResultTimeSec, &awResultDistKm, &awResultHR, &awResultFeeling,
			&awDueDate, &awCreatedAt,
		); err != nil {
			return nil, err
		}

		if seen[studentID] {
			continue
		}
		seen[studentID] = true

		item := models.DailySummaryItem{
			StudentID:   studentID,
			StudentName: studentName,
		}
		if studentAvatar.Valid && studentAvatar.String != "" {
			item.StudentAvatar = &studentAvatar.String
		}

		if awID.Valid {
			ws := &models.DailySummaryWorkout{
				ID:              awID.Int64,
				Title:           awTitle.String,
				Type:            awType.String,
				DistanceKm:      awDist.Float64,
				DurationSeconds: int(awDur.Int64),
				Description:     awDesc.String,
				Notes:           awNotes.String,
				Status:          awStatus.String,
				Segments:        []models.WorkoutSegment{},
			}
			if awResultTimeSec.Valid {
				v := int(awResultTimeSec.Int64)
				ws.ResultTimeSec = &v
			}
			if awResultDistKm.Valid {
				ws.ResultDistKm = &awResultDistKm.Float64
			}
			if awResultHR.Valid {
				v := int(awResultHR.Int64)
				ws.ResultHR = &v
			}
			if awResultFeeling.Valid {
				v := int(awResultFeeling.Int64)
				ws.ResultFeeling = &v
			}
			if awDueDate.Valid {
				ws.DueDate = awDueDate.String
			}
			item.Workout = ws
		}

		items = append(items, item)
	}

	// Load segments for all workouts
	for i, item := range items {
		if item.Workout != nil {
			items[i].Workout.Segments = r.FetchSegments(item.Workout.ID)
		}
	}

	return items, nil
}

func (r *coachRepository) GetUserName(id int64) (string, error) {
	var name string
	err := r.db.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", id).Scan(&name)
	return name, err
}

func (r *coachRepository) FetchSegments(awID int64) []models.WorkoutSegment {
	rows, err := r.db.Query(`
		SELECT id, assigned_workout_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM assigned_workout_segments
		WHERE assigned_workout_id = ?
		ORDER BY order_index ASC
	`, awID)
	if err != nil {
		return []models.WorkoutSegment{}
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			continue
		}
		segments = append(segments, s)
	}
	return segments
}

func (r *coachRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}
