package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type NotificationHandler struct {
	svc *services.NotificationService
}

func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// CreateNotification is a helper called by other handlers to emit notifications.
// Delegates to the service which checks notification preferences.
func (h *NotificationHandler) CreateNotification(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	return h.svc.Create(userID, notifType, title, body, metadata, actions)
}

// AcceptInvitation is called by InvitationHandler to accept an invitation.
func (h *NotificationHandler) AcceptInvitation(invitationID, userID int64) error {
	return h.svc.AcceptInvitation(invitationID, userID)
}

// RejectInvitation is called by InvitationHandler to reject an invitation.
func (h *NotificationHandler) RejectInvitation(invitationID, userID int64) {
	h.svc.RejectInvitation(invitationID, userID)
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

	notifications, err := h.svc.List(userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}

	writeJSON(w, http.StatusOK, notifications)
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	count, err := h.svc.UnreadCount(userID)
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

	found, err := h.svc.MarkRead(notifID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}
	if !found {
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

	if err := h.svc.MarkAllRead(userID); err != nil {
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

	err = h.svc.ExecuteAction(notifID, userID, req.Action)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Action executed"})
		return
	}

	switch err {
	case services.ErrNotFound:
		writeError(w, http.StatusNotFound, "Notification not found")
	case services.ErrInvitationNotPending:
		writeError(w, http.StatusConflict, "Invitation is no longer pending")
	case services.ErrStudentMaxCoaches:
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	prefs, err := h.svc.GetPreferences(userID)
	if err != nil {
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

	if err := h.svc.UpdatePreferences(userID, req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Preferences updated"})
}
