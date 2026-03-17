package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/fitreg/api/models"
)

type notificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	// Check preferences for configurable types
	if notifType == "workout_assigned" || notifType == "workout_completed" || notifType == "workout_skipped" || notifType == "assignment_message" {
		var workoutAssigned, workoutCompletedOrSkipped, assignmentMessage bool
		err := r.db.QueryRow(
			"SELECT COALESCE(workout_assigned, TRUE), COALESCE(workout_completed_or_skipped, TRUE), COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?",
			userID,
		).Scan(&workoutAssigned, &workoutCompletedOrSkipped, &assignmentMessage)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			workoutAssigned = true
			workoutCompletedOrSkipped = true
			assignmentMessage = true
		}
		if notifType == "workout_assigned" && !workoutAssigned {
			return nil
		}
		if (notifType == "workout_completed" || notifType == "workout_skipped") && !workoutCompletedOrSkipped {
			return nil
		}
		if notifType == "assignment_message" && !assignmentMessage {
			return nil
		}
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	var actionsJSON []byte
	if actions != nil {
		actionsJSON, err = json.Marshal(actions)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Exec(`
		INSERT INTO notifications (user_id, type, title, body, metadata, actions)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, notifType, title, body, metaJSON, actionsJSON)
	return err
}

func (r *notificationRepository) List(userID int64, limit, offset int) ([]models.Notification, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		var metadata, actions sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &actions, &n.IsRead, &n.CreatedAt); err != nil {
			continue
		}
		if metadata.Valid {
			n.Metadata = json.RawMessage(metadata.String)
		}
		if actions.Valid {
			n.Actions = json.RawMessage(actions.String)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *notificationRepository) UnreadCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = FALSE", userID).Scan(&count)
	return count, err
}

func (r *notificationRepository) MarkRead(notifID, userID int64) (bool, error) {
	result, err := r.db.Exec("UPDATE notifications SET is_read = TRUE WHERE id = ? AND user_id = ?", notifID, userID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *notificationRepository) MarkAllRead(userID int64) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = TRUE WHERE user_id = ? AND is_read = FALSE", userID)
	return err
}

func (r *notificationRepository) GetByID(notifID, userID int64) (models.Notification, error) {
	var n models.Notification
	var metadata, actions sql.NullString
	err := r.db.QueryRow(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications WHERE id = ? AND user_id = ?
	`, notifID, userID).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &actions, &n.IsRead, &n.CreatedAt)
	if err != nil {
		return n, err
	}
	if metadata.Valid {
		n.Metadata = json.RawMessage(metadata.String)
	}
	if actions.Valid {
		n.Actions = json.RawMessage(actions.String)
	}
	return n, nil
}

func (r *notificationRepository) ClearActions(notifID int64) error {
	_, err := r.db.Exec("UPDATE notifications SET actions = NULL WHERE id = ?", notifID)
	return err
}

func (r *notificationRepository) ClearActionsByInvitation(userID, invID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL
		WHERE type = 'invitation_received' AND user_id = ?
		AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, userID, invID)
	return err
}

func (r *notificationRepository) ClearCancelledInvitation(receiverID, invID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL, body = 'invitation_cancelled', is_read = TRUE
		WHERE type = 'invitation_received' AND user_id = ?
		AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, receiverID, invID)
	return err
}

func (r *notificationRepository) ClearCoachRequestActions(requesterID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, requesterID)
	return err
}

func (r *notificationRepository) GetPreferences(userID int64) (models.NotificationPreferences, error) {
	var prefs models.NotificationPreferences
	err := r.db.QueryRow(
		"SELECT id, user_id, workout_assigned, workout_completed_or_skipped, COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?",
		userID,
	).Scan(&prefs.ID, &prefs.UserID, &prefs.WorkoutAssigned, &prefs.WorkoutCompletedOrSkipped, &prefs.AssignmentMessage)
	if err == sql.ErrNoRows {
		return models.NotificationPreferences{
			UserID:                    userID,
			WorkoutAssigned:           true,
			WorkoutCompletedOrSkipped: true,
			AssignmentMessage:         true,
		}, nil
	}
	return prefs, err
}

func (r *notificationRepository) UpsertPreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error {
	_, err := r.db.Exec(`
		INSERT INTO notification_preferences (user_id, workout_assigned, workout_completed_or_skipped, assignment_message)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			workout_assigned = VALUES(workout_assigned),
			workout_completed_or_skipped = VALUES(workout_completed_or_skipped),
			assignment_message = VALUES(assignment_message)
	`, userID, req.WorkoutAssigned, req.WorkoutCompletedOrSkipped, req.AssignmentMessage)
	return err
}
