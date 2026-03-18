package repository

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/fitreg/api/models"
)

type assignmentMessageRepository struct {
	db *sql.DB
}

func NewAssignmentMessageRepository(db *sql.DB) AssignmentMessageRepository {
	return &assignmentMessageRepository{db: db}
}

func (r *assignmentMessageRepository) GetParticipants(awID int64) (int64, int64, string, string, error) {
	var coachID, studentID int64
	var status, title string
	err := r.db.QueryRow(
		"SELECT coach_id, student_id, status, title FROM assigned_workouts WHERE id = ?", awID,
	).Scan(&coachID, &studentID, &status, &title)
	return coachID, studentID, status, title, err
}

func (r *assignmentMessageRepository) List(awID int64) ([]models.AssignmentMessage, error) {
	rows, err := r.db.Query(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.assigned_workout_id = ?
		ORDER BY am.created_at ASC
	`, awID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []models.AssignmentMessage{}
	for rows.Next() {
		var m models.AssignmentMessage
		var avatar sql.NullString
		if err := rows.Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
			&m.Body, &m.IsRead, &m.CreatedAt); err != nil {
			return nil, err
		}
		if avatar.Valid {
			m.SenderAvatar = avatar.String
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *assignmentMessageRepository) Create(awID, senderID int64, body string) (models.AssignmentMessage, error) {
	result, err := r.db.Exec(
		"INSERT INTO assignment_messages (assigned_workout_id, sender_id, body) VALUES (?, ?, ?)",
		awID, senderID, body,
	)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	msgID, _ := result.LastInsertId()
	var m models.AssignmentMessage
	var avatar sql.NullString
	err = r.db.QueryRow(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.id = ?
	`, msgID).Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
		&m.Body, &m.IsRead, &m.CreatedAt)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	if avatar.Valid {
		m.SenderAvatar = avatar.String
	}
	return m, nil
}

func (r *assignmentMessageRepository) MarkRead(awID, userID int64) error {
	_, err := r.db.Exec(
		"UPDATE assignment_messages SET is_read = TRUE WHERE assigned_workout_id = ? AND sender_id != ?",
		awID, userID,
	)
	return err
}

func (r *assignmentMessageRepository) GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error) {
	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields, workoutType sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds sql.NullInt64
	var studentName, coachName string
	err := r.db.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at,
			us.name AS student_name, uc.name AS coach_name,
			(SELECT COUNT(*) FROM assignment_messages am
				WHERE am.assigned_workout_id = aw.id AND am.sender_id != ? AND am.is_read = FALSE) AS unread_message_count
		FROM assigned_workouts aw
		JOIN users us ON us.id = aw.student_id
		JOIN users uc ON uc.id = aw.coach_id
		WHERE aw.id = ? AND (aw.coach_id = ? OR aw.student_id = ?)
	`, userID, awID, userID, userID).Scan(
		&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &workoutType,
		&distanceKm, &durationSeconds, &notes, &expectedFields,
		&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
		&aw.ImageFileID, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt,
		&studentName, &coachName, &aw.UnreadMessageCount,
	)
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
	aw.CoachName = coachName
	return aw, nil
}

func (r *assignmentMessageRepository) FetchSegments(awID int64) []models.WorkoutSegment {
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

func (r *assignmentMessageRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}

// truncateDate trims a datetime string to date only (first 10 chars).
// Used by GetAssignedWorkoutDetail and coach_repository.go (same package).
func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// escapeLike escapes LIKE wildcard characters (%, _, \) in user-supplied
// search terms so they are treated as literals, not SQL wildcards.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
