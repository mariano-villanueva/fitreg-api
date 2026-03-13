package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type NotificationHandler struct {
	DB *sql.DB
}

func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{DB: db}
}

// CreateNotification is a helper called by other handlers to emit notifications.
// It checks notification preferences before creating.
func (h *NotificationHandler) CreateNotification(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	// Check preferences for configurable types
	if notifType == "workout_assigned" || notifType == "workout_completed" || notifType == "workout_skipped" || notifType == "assignment_message" {
		var workoutAssigned, workoutCompletedOrSkipped, assignmentMessage bool
		err := h.DB.QueryRow("SELECT COALESCE(workout_assigned, TRUE), COALESCE(workout_completed_or_skipped, TRUE), COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?", userID).Scan(&workoutAssigned, &workoutCompletedOrSkipped, &assignmentMessage)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		// If no row found, defaults are true
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

	_, err = h.DB.Exec(`
		INSERT INTO notifications (user_id, type, title, body, metadata, actions)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, notifType, title, body, metaJSON, actionsJSON)
	return err
}

func (h *NotificationHandler) isAdmin(userID int64) bool {
	var isAdmin bool
	err := h.DB.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return err == nil && isAdmin
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := h.DB.Query(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		var metadata, actions sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &actions, &n.IsRead, &n.CreatedAt); err != nil {
			log.Printf("ERROR scanning notification: %v", err)
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

	writeJSON(w, http.StatusOK, notifications)
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = FALSE", userID).Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count notifications")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/read")
	notifID, err := extractID(path, "/api/notifications/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	result, err := h.DB.Exec("UPDATE notifications SET is_read = TRUE WHERE id = ? AND user_id = ?", notifID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for mark read", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	_, err := h.DB.Exec("UPDATE notifications SET is_read = TRUE WHERE user_id = ? AND is_read = FALSE", userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark notifications as read")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/action")
	notifID, err := extractID(path, "/api/notifications/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}

	var req models.NotificationActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Fetch notification
	var notif models.Notification
	var metadata, actions sql.NullString
	err = h.DB.QueryRow(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications WHERE id = ? AND user_id = ?
	`, notifID, userID).Scan(&notif.ID, &notif.UserID, &notif.Type, &notif.Title, &notif.Body, &metadata, &actions, &notif.IsRead, &notif.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch notification")
		return
	}

	if !actions.Valid || actions.String == "" || actions.String == "null" {
		writeError(w, http.StatusBadRequest, "No actions available for this notification")
		return
	}

	// Validate action key exists
	var actionList []models.NotificationAction
	if err := json.Unmarshal([]byte(actions.String), &actionList); err != nil {
		writeError(w, http.StatusInternalServerError, "Invalid actions data")
		return
	}
	validAction := false
	for _, a := range actionList {
		if a.Key == req.Action {
			validAction = true
			break
		}
	}
	if !validAction {
		writeError(w, http.StatusBadRequest, "Invalid action")
		return
	}

	// Resolve action based on notification type
	switch notif.Type {
	case "invitation_received":
		var meta struct {
			InvitationID int64 `json:"invitation_id"`
		}
		if metadata.Valid {
			if err := json.Unmarshal([]byte(metadata.String), &meta); err != nil {
				logErr("unmarshal invitation metadata", err)
			}
		}
		if meta.InvitationID == 0 {
			writeError(w, http.StatusInternalServerError, "Missing invitation reference")
			return
		}

		// Check invitation is still pending
		var invStatus string
		err := h.DB.QueryRow("SELECT status FROM invitations WHERE id = ?", meta.InvitationID).Scan(&invStatus)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to check invitation")
			return
		}
		if invStatus != "pending" {
			if _, err := h.DB.Exec("UPDATE notifications SET actions = NULL WHERE id = ?", notifID); err != nil {
				logErr("clear actions for non-pending invitation", err)
			}
			writeError(w, http.StatusConflict, "Invitation is no longer pending")
			return
		}

		switch req.Action {
		case "accept":
			if err := h.acceptInvitation(meta.InvitationID, userID); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		case "reject":
			h.rejectInvitation(meta.InvitationID, userID)
		}

	case "coach_request":
		var meta struct {
			RequesterID   int64  `json:"requester_id"`
			RequesterName string `json:"requester_name"`
		}
		if metadata.Valid {
			if err := json.Unmarshal([]byte(metadata.String), &meta); err != nil {
				logErr("unmarshal coach request metadata", err)
			}
		}
		if meta.RequesterID == 0 {
			writeError(w, http.StatusInternalServerError, "Missing requester reference")
			return
		}

		switch req.Action {
		case "approve":
			h.approveCoachRequest(meta.RequesterID, meta.RequesterName)
		case "reject":
			h.rejectCoachRequest(meta.RequesterID, meta.RequesterName)
		}

		// Clear actions on ALL coach_request notifications for this requester
		if _, err := h.DB.Exec(`
			UPDATE notifications SET actions = NULL
			WHERE type = 'coach_request' AND actions IS NOT NULL
			AND JSON_EXTRACT(metadata, '$.requester_id') = ?
		`, meta.RequesterID); err != nil {
			logErr("clear coach request notification actions", err)
		}

	default:
		writeError(w, http.StatusBadRequest, "Unsupported notification type for actions")
		return
	}

	// Clear actions after execution
	if _, err := h.DB.Exec("UPDATE notifications SET actions = NULL WHERE id = ?", notifID); err != nil {
		logErr("clear notification actions after execution", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Action executed"})
}

// acceptInvitation handles the accept logic for an invitation.
// Uses transaction with SELECT FOR UPDATE to prevent race conditions.
func (h *NotificationHandler) acceptInvitation(invitationID, userID int64) error {
	tx, err := h.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lock and fetch invitation
	var invType string
	var senderID, receiverID int64
	err = tx.QueryRow("SELECT type, sender_id, receiver_id FROM invitations WHERE id = ? AND status = 'pending' FOR UPDATE", invitationID).Scan(&invType, &senderID, &receiverID)
	if err != nil {
		return fmt.Errorf("invitation not found or already resolved")
	}

	// Determine coach and student
	var coachID, studentID int64
	if invType == "coach_invite" {
		coachID = senderID
		studentID = receiverID
	} else {
		coachID = receiverID
		studentID = senderID
	}

	// Check MaxCoachesPerStudent
	var activeCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active' FOR UPDATE", studentID).Scan(&activeCount); err != nil {
		logErr("count active coaches for student", err)
	}
	if activeCount >= models.MaxCoachesPerStudent {
		return fmt.Errorf("student has reached the maximum number of coaches (%d)", models.MaxCoachesPerStudent)
	}

	// Create coach_students record
	_, err = tx.Exec(`
		INSERT INTO coach_students (coach_id, student_id, invitation_id, status, started_at)
		VALUES (?, ?, ?, 'active', NOW())
	`, coachID, studentID, invitationID)
	if err != nil {
		return fmt.Errorf("failed to create relationship")
	}

	// Update invitation status
	_, err = tx.Exec("UPDATE invitations SET status = 'accepted', updated_at = NOW() WHERE id = ?", invitationID)
	if err != nil {
		return fmt.Errorf("failed to update invitation")
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Create notification for sender (outside transaction)
	var acceptorName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", userID).Scan(&acceptorName); err != nil {
		logErr("fetch acceptor name", err)
	}
	meta := map[string]interface{}{"invitation_id": invitationID, "user_name": acceptorName}
	h.CreateNotification(senderID, "invitation_accepted", "notif_invitation_accepted_title", "notif_invitation_accepted_body", meta, nil)

	return nil
}

func (h *NotificationHandler) rejectInvitation(invitationID, userID int64) {
	if _, err := h.DB.Exec("UPDATE invitations SET status = 'rejected', updated_at = NOW() WHERE id = ?", invitationID); err != nil {
		logErr("reject invitation update", err)
	}

	var senderID int64
	if err := h.DB.QueryRow("SELECT sender_id FROM invitations WHERE id = ?", invitationID).Scan(&senderID); err != nil {
		logErr("fetch sender id for rejection", err)
	}

	var userName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", userID).Scan(&userName); err != nil {
		logErr("fetch user name for rejection", err)
	}
	meta := map[string]interface{}{"invitation_id": invitationID, "user_name": userName}
	h.CreateNotification(senderID, "invitation_rejected", "notif_invitation_rejected_title", "notif_invitation_rejected_body", meta, nil)
}

func (h *NotificationHandler) approveCoachRequest(requesterID int64, requesterName string) {
	// Set user as coach and make profile public
	if _, err := h.DB.Exec("UPDATE users SET is_coach = TRUE, coach_public = TRUE, updated_at = NOW() WHERE id = ?", requesterID); err != nil {
		logErr("approve coach request update", err)
	}

	// Notify requester
	meta := map[string]interface{}{"requester_name": requesterName}
	h.CreateNotification(requesterID, "coach_request_approved",
		"notif_coach_request_approved_title", "notif_coach_request_approved_body",
		meta, nil)
}

func (h *NotificationHandler) rejectCoachRequest(requesterID int64, requesterName string) {
	// Notify requester
	meta := map[string]interface{}{"requester_name": requesterName}
	h.CreateNotification(requesterID, "coach_request_rejected",
		"notif_coach_request_rejected_title", "notif_coach_request_rejected_body",
		meta, nil)
}

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var prefs models.NotificationPreferences
	err := h.DB.QueryRow("SELECT id, user_id, workout_assigned, workout_completed_or_skipped, COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?", userID).Scan(&prefs.ID, &prefs.UserID, &prefs.WorkoutAssigned, &prefs.WorkoutCompletedOrSkipped, &prefs.AssignmentMessage)
	if err == sql.ErrNoRows {
		prefs = models.NotificationPreferences{UserID: userID, WorkoutAssigned: true, WorkoutCompletedOrSkipped: true, AssignmentMessage: true}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch preferences")
		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateNotificationPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO notification_preferences (user_id, workout_assigned, workout_completed_or_skipped, assignment_message)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE workout_assigned = VALUES(workout_assigned), workout_completed_or_skipped = VALUES(workout_completed_or_skipped), assignment_message = VALUES(assignment_message)
	`, userID, req.WorkoutAssigned, req.WorkoutCompletedOrSkipped, req.AssignmentMessage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Preferences updated"})
}
