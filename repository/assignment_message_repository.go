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

func (r *assignmentMessageRepository) GetParticipants(workoutID int64) (int64, int64, string, string, error) {
	var coachID, studentID int64
	var status, title string
	err := r.db.QueryRow(
		"SELECT coach_id, user_id, status, COALESCE(title,'') FROM workouts WHERE id = ?", workoutID,
	).Scan(&coachID, &studentID, &status, &title)
	return coachID, studentID, status, title, err
}

func (r *assignmentMessageRepository) List(workoutID int64) ([]models.AssignmentMessage, error) {
	rows, err := r.db.Query(`
		SELECT am.id, am.workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.workout_id = ?
		ORDER BY am.created_at ASC
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []models.AssignmentMessage{}
	for rows.Next() {
		var m models.AssignmentMessage
		var avatar sql.NullString
		if err := rows.Scan(&m.ID, &m.WorkoutID, &m.SenderID, &m.SenderName, &avatar,
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

func (r *assignmentMessageRepository) Create(workoutID, senderID int64, body string) (models.AssignmentMessage, error) {
	result, err := r.db.Exec(
		"INSERT INTO assignment_messages (workout_id, sender_id, body) VALUES (?, ?, ?)",
		workoutID, senderID, body,
	)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	msgID, _ := result.LastInsertId()
	var m models.AssignmentMessage
	var avatar sql.NullString
	err = r.db.QueryRow(`
		SELECT am.id, am.workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.id = ?
	`, msgID).Scan(&m.ID, &m.WorkoutID, &m.SenderID, &m.SenderName, &avatar,
		&m.Body, &m.IsRead, &m.CreatedAt)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	if avatar.Valid {
		m.SenderAvatar = avatar.String
	}
	return m, nil
}

func (r *assignmentMessageRepository) MarkRead(workoutID, userID int64) error {
	_, err := r.db.Exec(
		"UPDATE assignment_messages SET is_read = TRUE WHERE workout_id = ? AND sender_id != ?",
		workoutID, userID,
	)
	return err
}

func (r *assignmentMessageRepository) GetWorkoutDetail(workoutID, userID int64) (models.Workout, error) {
	var wo models.Workout
	var coachID sql.NullInt64
	var title, description, workoutType, notes, avgPace sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds, calories sql.NullInt64
	var expectedFields sql.NullString
	var resultDistKm sql.NullFloat64
	var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
	var dueDate sql.NullString
	var studentName, coachName string
	var unread int

	err := r.db.QueryRow(`
		SELECT w.id, w.user_id, w.coach_id, w.title, w.description, w.type,
			w.distance_km, w.duration_seconds, w.notes, w.expected_fields,
			w.result_time_seconds, w.result_distance_km, w.result_heart_rate, w.result_feeling,
			w.image_file_id, w.status, w.due_date, w.avg_pace, w.calories,
			w.created_at, w.updated_at,
			us.name AS student_name, uc.name AS coach_name,
			(SELECT COUNT(*) FROM assignment_messages am
				WHERE am.workout_id = w.id AND am.sender_id != ? AND am.is_read = FALSE) AS unread_count
		FROM workouts w
		JOIN users us ON us.id = w.user_id
		JOIN users uc ON uc.id = w.coach_id
		WHERE w.id = ? AND w.coach_id IS NOT NULL AND (w.coach_id = ? OR w.user_id = ?)
	`, userID, workoutID, userID, userID).Scan(
		&wo.ID, &wo.UserID, &coachID,
		&title, &description, &workoutType,
		&distanceKm, &durationSeconds, &notes, &expectedFields,
		&resultTimeSec, &resultDistKm, &resultHR, &resultFeeling,
		&imageFileID, &wo.Status, &dueDate, &avgPace, &calories,
		&wo.CreatedAt, &wo.UpdatedAt,
		&studentName, &coachName, &unread,
	)
	if err != nil {
		return models.Workout{}, err
	}
	if coachID.Valid {
		v := coachID.Int64
		wo.CoachID = &v
	}
	if title.Valid { wo.Title = title.String }
	if description.Valid { wo.Description = description.String }
	if workoutType.Valid { wo.Type = workoutType.String }
	if notes.Valid { wo.Notes = notes.String }
	if dueDate.Valid { wo.DueDate = truncateDate(dueDate.String) }
	if distanceKm.Valid { wo.DistanceKm = distanceKm.Float64 }
	if durationSeconds.Valid { wo.DurationSeconds = int(durationSeconds.Int64) }
	if expectedFields.Valid { wo.ExpectedFields = json.RawMessage(expectedFields.String) }
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
	if avgPace.Valid { wo.AvgPace = avgPace.String }
	if calories.Valid { wo.Calories = int(calories.Int64) }
	if imageFileID.Valid {
		v := imageFileID.Int64
		wo.ImageFileID = &v
	}
	wo.UserName = studentName
	wo.CoachName = coachName
	wo.UnreadMessageCount = unread
	return wo, nil
}

func (r *assignmentMessageRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}

// truncateDate trims a datetime string to date only (first 10 chars).
// Used by GetWorkoutDetail and coach_repository.go (same package).
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
