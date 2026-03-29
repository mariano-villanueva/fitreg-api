package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fitreg/api/models"
)

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
		SELECT id, user_id, coach_id, title, description, type, notes, due_date,
			distance_km, duration_seconds, expected_fields,
			result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
			avg_pace, calories, image_file_id, status, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY due_date DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var coachID sql.NullInt64
		var title, description, workoutType, notes, avgPace sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds, calories sql.NullInt64
		var expectedFields sql.NullString
		var resultDistKm sql.NullFloat64
		var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
		var dueDate sql.NullString
		if err := rows.Scan(
			&wo.ID, &wo.UserID, &coachID,
			&title, &description, &workoutType, &notes, &dueDate,
			&distanceKm, &durationSeconds,
			&expectedFields,
			&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
			&avgPace, &calories, &imageFileID,
			&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
		); err != nil {
			return nil, err
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
		workouts = append(workouts, wo)
	}
	return workouts, rows.Err()
}

func (r *coachRepository) GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error) {
	rows, err := r.db.Query(`
		SELECT
			u.id, u.name,
			CASE WHEN u.custom_avatar IS NOT NULL AND u.custom_avatar != '' THEN u.custom_avatar ELSE u.avatar_url END as avatar,
			w.id, w.title, COALESCE(w.type, ''), w.distance_km, w.duration_seconds,
			COALESCE(w.description, ''), COALESCE(w.notes, ''), w.status,
			w.result_time_seconds, w.result_distance_km, w.result_heart_rate, w.result_feeling,
			w.due_date, w.created_at
		FROM coach_students cs
		JOIN users u ON u.id = cs.student_id
		LEFT JOIN workouts w
			ON w.user_id = cs.student_id
			AND w.coach_id = ?
			AND w.due_date = ?
		WHERE cs.coach_id = ? AND cs.status = 'active'
		ORDER BY u.name ASC, w.created_at DESC
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

	// Load segments for all workouts (skip for callers that don't need them, e.g. compliance dashboard)
	if includeSegments {
		for i, item := range items {
			if item.Workout != nil {
				segRows, err := r.db.Query(`
					SELECT id, workout_id, order_index, segment_type,
						COALESCE(repetitions,1), COALESCE(value,0), COALESCE(unit,''), COALESCE(intensity,''),
						COALESCE(work_value,0), COALESCE(work_unit,''), COALESCE(work_intensity,''),
						COALESCE(rest_value,0), COALESCE(rest_unit,''), COALESCE(rest_intensity,'')
					FROM workout_segments WHERE workout_id = ? ORDER BY order_index
				`, item.Workout.ID)
				if err == nil {
					segs := []models.WorkoutSegment{}
					for segRows.Next() {
						var s models.WorkoutSegment
						if err := segRows.Scan(&s.ID, &s.WorkoutID, &s.OrderIndex, &s.SegmentType,
							&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
							&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
							&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
							continue
						}
						segs = append(segs, s)
					}
					segRows.Close()
					items[i].Workout.Segments = segs
				}
			}
		}
	}

	return items, nil
}

func (r *coachRepository) GetUserName(id int64) (string, error) {
	var name string
	err := r.db.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", id).Scan(&name)
	return name, err
}

func (r *coachRepository) GetWeeklyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -weeks*7).Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT
			DATE_FORMAT(DATE_SUB(due_date, INTERVAL WEEKDAY(due_date) DAY), '%Y-%m-%d') AS week_start,
			SUM(CASE WHEN coach_id IS NOT NULL AND status = 'pending' THEN COALESCE(distance_km, 0) ELSE 0 END) AS planned_km,
			SUM(CASE WHEN status = 'completed' THEN COALESCE(result_distance_km, distance_km, 0) ELSE 0 END) AS actual_km,
			SUM(CASE WHEN coach_id IS NOT NULL AND status = 'pending' THEN COALESCE(duration_seconds, 0) ELSE 0 END) AS planned_seconds,
			SUM(CASE WHEN status = 'completed' THEN COALESCE(result_time_seconds, duration_seconds, 0) ELSE 0 END) AS actual_seconds,
			COUNT(CASE WHEN coach_id IS NOT NULL THEN 1 END) AS sessions_planned,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) AS sessions_completed,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) AS sessions_skipped,
			MAX(CASE WHEN coach_id IS NULL THEN 1 ELSE 0 END) AS has_personal_workouts
		FROM workouts
		WHERE user_id = ? AND due_date >= ?
		GROUP BY week_start
		ORDER BY week_start ASC
	`, studentID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []models.WeeklyLoadEntry{}
	for rows.Next() {
		var e models.WeeklyLoadEntry
		if err := rows.Scan(
			&e.WeekStart, &e.PlannedKm, &e.ActualKm,
			&e.PlannedSeconds, &e.ActualSeconds,
			&e.SessionsPlanned, &e.SessionsCompleted, &e.SessionsSkipped,
			&e.HasPersonalWorkouts,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
