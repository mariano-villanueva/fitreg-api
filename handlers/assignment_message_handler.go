package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type AssignmentMessageHandler struct {
	DB           *sql.DB
	Notification *NotificationHandler
}

func NewAssignmentMessageHandler(db *sql.DB, nh *NotificationHandler) *AssignmentMessageHandler {
	return &AssignmentMessageHandler{DB: db, Notification: nh}
}

func (h *AssignmentMessageHandler) getAssignmentParticipants(awID int64) (int64, int64, string, string, error) {
	var coachID, studentID int64
	var status, title string
	err := h.DB.QueryRow(
		"SELECT coach_id, student_id, status, title FROM assigned_workouts WHERE id = ?", awID,
	).Scan(&coachID, &studentID, &status, &title)
	return coachID, studentID, status, title, err
}

// ListMessages handles GET /api/assignment-messages/{id}
func (h *AssignmentMessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/assignment-messages/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	coachID, studentID, _, _, err := h.getAssignmentParticipants(awID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workout")
		return
	}

	if userID != coachID && userID != studentID {
		writeError(w, http.StatusForbidden, "Forbidden")
		return
	}

	rows, err := h.DB.Query(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.assigned_workout_id = ?
		ORDER BY am.created_at ASC
	`, awID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}
	defer rows.Close()

	messages := []models.AssignmentMessage{}
	for rows.Next() {
		var m models.AssignmentMessage
		var avatar sql.NullString
		if err := rows.Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
			&m.Body, &m.IsRead, &m.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to scan message")
			return
		}
		if avatar.Valid {
			m.SenderAvatar = avatar.String
		}
		messages = append(messages, m)
	}

	writeJSON(w, http.StatusOK, messages)
}

// SendMessage handles POST /api/assignment-messages/{id}
func (h *AssignmentMessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/assignment-messages/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	coachID, studentID, status, title, err := h.getAssignmentParticipants(awID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workout")
		return
	}

	if userID != coachID && userID != studentID {
		writeError(w, http.StatusForbidden, "Forbidden")
		return
	}

	if status != "pending" {
		writeError(w, http.StatusConflict, "Cannot send messages on a non-pending assignment")
		return
	}

	var req models.CreateAssignmentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	body := strings.TrimSpace(req.Body)
	charCount := utf8.RuneCountInString(body)
	if charCount < 1 || charCount > 2000 {
		writeError(w, http.StatusBadRequest, "Message body must be between 1 and 2000 characters")
		return
	}

	result, err := h.DB.Exec(`
		INSERT INTO assignment_messages (assigned_workout_id, sender_id, body)
		VALUES (?, ?, ?)
	`, awID, userID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to send message")
		return
	}

	msgID, _ := result.LastInsertId()

	var m models.AssignmentMessage
	var avatar sql.NullString
	err = h.DB.QueryRow(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.id = ?
	`, msgID).Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
		&m.Body, &m.IsRead, &m.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch created message")
		return
	}
	if avatar.Valid {
		m.SenderAvatar = avatar.String
	}

	// Notify the other participant
	var recipientID int64
	if userID == coachID {
		recipientID = studentID
	} else {
		recipientID = coachID
	}

	notifMeta := map[string]interface{}{
		"assigned_workout_id": awID,
		"workout_title":      title,
		"sender_id":          userID,
		"sender_name":        m.SenderName,
	}
	err = h.Notification.CreateNotification(recipientID, "assignment_message",
		"notif_assignment_message_title", "notif_assignment_message_body", notifMeta, nil)
	if err != nil {
		logErr("CreateNotification for assignment_message", err)
	}

	writeJSON(w, http.StatusCreated, m)
}

// MarkRead handles PUT /api/assignment-messages/{id}/read
func (h *AssignmentMessageHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(strings.TrimSuffix(r.URL.Path, "/read"), "/api/assignment-messages/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	coachID, studentID, _, _, err := h.getAssignmentParticipants(awID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workout")
		return
	}

	if userID != coachID && userID != studentID {
		writeError(w, http.StatusForbidden, "Forbidden")
		return
	}

	_, err = h.DB.Exec(`
		UPDATE assignment_messages SET is_read = TRUE
		WHERE assigned_workout_id = ? AND sender_id != ?
	`, awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark messages as read")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetAssignedWorkoutDetail handles GET /api/assigned-workout-detail/{id}
func (h *AssignmentMessageHandler) GetAssignedWorkoutDetail(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/assigned-workout-detail/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields sql.NullString
	var studentName, coachName string
	err = h.DB.QueryRow(`
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
		&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
		&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
		&aw.ImageFileID, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt,
		&studentName, &coachName, &aw.UnreadMessageCount,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		logErr("GetAssignedWorkoutDetail", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch assigned workout: %v", err))
		return
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
	aw.Segments = fetchSegments(h.DB, aw.ID)

	if aw.ImageFileID != nil {
		var uuid string
		if err := h.DB.QueryRow("SELECT uuid FROM files WHERE id = ?", *aw.ImageFileID).Scan(&uuid); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}

	writeJSON(w, http.StatusOK, aw)
}
