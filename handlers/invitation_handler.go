package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type InvitationHandler struct {
	DB           *sql.DB
	Notification *NotificationHandler
}

func NewInvitationHandler(db *sql.DB, nh *NotificationHandler) *InvitationHandler {
	return &InvitationHandler{DB: db, Notification: nh}
}

func (h *InvitationHandler) isAdmin(userID int64) bool {
	var isAdmin bool
	err := h.DB.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return err == nil && isAdmin
}

func (h *InvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find receiver
	var receiverID int64
	var receiverIsCoach, receiverCoachPublic bool
	if req.ReceiverID > 0 {
		receiverID = req.ReceiverID
		err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE id = ?", receiverID).Scan(&receiverIsCoach, &receiverCoachPublic)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Cannot send invitation")
			return
		}
	} else {
		err := h.DB.QueryRow("SELECT id, COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE email = ?", req.ReceiverEmail).Scan(&receiverID, &receiverIsCoach, &receiverCoachPublic)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Cannot send invitation")
			return
		}
	}

	// No self-invitation
	if userID == receiverID {
		writeError(w, http.StatusBadRequest, "Cannot send invitation")
		return
	}

	// Validate type-specific rules
	if req.Type == "coach_invite" {
		var isCoach bool
		h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
		if !isCoach {
			writeError(w, http.StatusBadRequest, "Cannot send invitation")
			return
		}
	} else if req.Type == "student_request" {
		if !receiverIsCoach || !receiverCoachPublic {
			writeError(w, http.StatusBadRequest, "Cannot send invitation")
			return
		}
	} else {
		writeError(w, http.StatusBadRequest, "Invalid invitation type")
		return
	}

	// Check no pending invitation exists between these users (either direction)
	var pendingCount int
	h.DB.QueryRow(`
		SELECT COUNT(*) FROM invitations
		WHERE status = 'pending' AND (
			(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		)
	`, userID, receiverID, receiverID, userID).Scan(&pendingCount)
	if pendingCount > 0 {
		writeError(w, http.StatusBadRequest, "Cannot send invitation")
		return
	}

	// Check no active relationship exists
	var activeCount int
	h.DB.QueryRow(`
		SELECT COUNT(*) FROM coach_students
		WHERE status = 'active' AND (
			(coach_id = ? AND student_id = ?) OR (coach_id = ? AND student_id = ?)
		)
	`, userID, receiverID, receiverID, userID).Scan(&activeCount)
	if activeCount > 0 {
		writeError(w, http.StatusBadRequest, "Cannot send invitation")
		return
	}

	// Check MaxCoachesPerStudent (early check)
	var studentID int64
	if req.Type == "coach_invite" {
		studentID = receiverID
	} else {
		studentID = userID
	}
	var studentCoachCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active'", studentID).Scan(&studentCoachCount)
	if studentCoachCount >= models.MaxCoachesPerStudent {
		writeError(w, http.StatusBadRequest, "Cannot send invitation")
		return
	}

	// Create invitation
	result, err := h.DB.Exec(`
		INSERT INTO invitations (type, sender_id, receiver_id, message, status)
		VALUES (?, ?, ?, ?, 'pending')
	`, req.Type, userID, receiverID, req.Message)
	if err != nil {
		log.Printf("ERROR creating invitation: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}
	invID, _ := result.LastInsertId()

	// Create notification for receiver
	var senderName, senderAvatar string
	h.DB.QueryRow("SELECT COALESCE(name, ''), COALESCE(avatar_url, '') FROM users WHERE id = ?", userID).Scan(&senderName, &senderAvatar)

	meta := map[string]interface{}{
		"invitation_id": invID,
		"sender_id":     userID,
		"sender_name":   senderName,
		"sender_avatar": senderAvatar,
	}
	actions := []models.NotificationAction{
		{Key: "accept", Label: "invitation_accept", Style: "primary"},
		{Key: "reject", Label: "invitation_reject", Style: "danger"},
	}

	var title, body string
	if req.Type == "coach_invite" {
		title = "notif_coach_invite_title"
		body = "notif_coach_invite_body"
	} else {
		title = "notif_student_request_title"
		body = "notif_student_request_body"
	}
	h.Notification.CreateNotification(receiverID, "invitation_received", title, body, meta, actions)

	// Return created invitation
	var inv models.Invitation
	h.DB.QueryRow(`
		SELECT i.id, i.type, i.sender_id, i.receiver_id, COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
			COALESCE(s.name, ''), COALESCE(s.avatar_url, ''), COALESCE(rv.name, ''), COALESCE(rv.avatar_url, '')
		FROM invitations i
		JOIN users s ON s.id = i.sender_id
		JOIN users rv ON rv.id = i.receiver_id
		WHERE i.id = ?
	`, invID).Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar)

	writeJSON(w, http.StatusCreated, inv)
}

func (h *InvitationHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	status := r.URL.Query().Get("status")
	direction := r.URL.Query().Get("direction")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := `
		SELECT i.id, i.type, i.sender_id, i.receiver_id, COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
			COALESCE(s.name, ''), COALESCE(s.avatar_url, ''), COALESCE(rv.name, ''), COALESCE(rv.avatar_url, '')
		FROM invitations i
		JOIN users s ON s.id = i.sender_id
		JOIN users rv ON rv.id = i.receiver_id
		WHERE 1=1
	`
	args := []interface{}{}

	if direction == "sent" {
		query += " AND i.sender_id = ?"
		args = append(args, userID)
	} else if direction == "received" {
		query += " AND i.receiver_id = ?"
		args = append(args, userID)
	} else {
		query += " AND (i.sender_id = ? OR i.receiver_id = ?)"
		args = append(args, userID, userID)
	}

	if status != "" {
		query += " AND i.status = ?"
		args = append(args, status)
	}

	query += " ORDER BY i.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Printf("ERROR listing invitations: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch invitations")
		return
	}
	defer rows.Close()

	invitations := []models.Invitation{}
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
			&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar); err != nil {
			log.Printf("ERROR scanning invitation: %v", err)
			continue
		}
		invitations = append(invitations, inv)
	}

	writeJSON(w, http.StatusOK, invitations)
}

func (h *InvitationHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	invID, err := extractID(r.URL.Path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	var inv models.Invitation
	err = h.DB.QueryRow(`
		SELECT i.id, i.type, i.sender_id, i.receiver_id, COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
			COALESCE(s.name, ''), COALESCE(s.avatar_url, ''), COALESCE(rv.name, ''), COALESCE(rv.avatar_url, '')
		FROM invitations i
		JOIN users s ON s.id = i.sender_id
		JOIN users rv ON rv.id = i.receiver_id
		WHERE i.id = ?
	`, invID).Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Invitation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch invitation")
		return
	}

	// Check ownership
	if inv.SenderID != userID && inv.ReceiverID != userID && !h.isAdmin(userID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	writeJSON(w, http.StatusOK, inv)
}

func (h *InvitationHandler) RespondInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/respond")
	invID, err := extractID(path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	var req models.RespondInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Action != "accepted" && req.Action != "rejected" {
		writeError(w, http.StatusBadRequest, "Action must be 'accepted' or 'rejected'")
		return
	}

	// Fetch invitation and verify receiver
	var invSenderID, invReceiverID int64
	var invStatus, invType string
	err = h.DB.QueryRow("SELECT sender_id, receiver_id, status, type FROM invitations WHERE id = ?", invID).Scan(&invSenderID, &invReceiverID, &invStatus, &invType)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Invitation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch invitation")
		return
	}

	if invReceiverID != userID && !h.isAdmin(userID) {
		writeError(w, http.StatusForbidden, "Only the receiver can respond")
		return
	}

	if invStatus != "pending" {
		writeError(w, http.StatusConflict, "Invitation is no longer pending")
		return
	}

	if req.Action == "accepted" {
		if err := h.Notification.acceptInvitation(invID, userID); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		h.Notification.rejectInvitation(invID, userID)
	}

	// Nullify actions on related notification
	h.DB.Exec(`
		UPDATE notifications SET actions = NULL
		WHERE type = 'invitation_received' AND user_id = ? AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, userID, invID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation " + req.Action})
}

func (h *InvitationHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	invID, err := extractID(r.URL.Path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	var senderID, receiverID int64
	var status string
	err = h.DB.QueryRow("SELECT sender_id, receiver_id, status FROM invitations WHERE id = ?", invID).Scan(&senderID, &receiverID, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Invitation not found")
		return
	}
	if senderID != userID && !h.isAdmin(userID) {
		writeError(w, http.StatusForbidden, "Only the sender can cancel")
		return
	}
	if status != "pending" {
		writeError(w, http.StatusConflict, "Invitation is no longer pending")
		return
	}

	h.DB.Exec("UPDATE invitations SET status = 'cancelled', updated_at = NOW() WHERE id = ?", invID)

	// Nullify actions and mark as read on related notification
	h.DB.Exec(`
		UPDATE notifications SET actions = NULL, body = 'invitation_cancelled', is_read = TRUE
		WHERE type = 'invitation_received' AND user_id = ? AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, receiverID, invID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation cancelled"})
}
