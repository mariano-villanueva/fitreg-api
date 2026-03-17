package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type UserHandler struct {
	svc *services.UserService
	nh  *NotificationHandler // interim: kept until Plan C migrates NotificationHandler
}

func NewUserHandler(svc *services.UserService, nh *NotificationHandler) *UserHandler {
	return &UserHandler{svc: svc, nh: nh}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	u, err := h.svc.GetProfile(userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	u, err := h.svc.UpdateProfile(userID, req)
	if err != nil {
		log.Printf("ERROR UpdateProfile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) RequestCoach(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Locality string   `json:"locality"`
		Level    []string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Level) == 0 {
		writeError(w, http.StatusBadRequest, "At least one level is required")
		return
	}

	isCoach, err := h.svc.IsCoach(userID)
	if err != nil {
		logErr("check is coach for request", err)
	}
	if isCoach {
		writeError(w, http.StatusConflict, "User is already a coach")
		return
	}

	pending, err := h.svc.HasPendingCoachRequest(userID)
	if err != nil {
		logErr("check pending coach request count", err)
	}
	if pending {
		writeError(w, http.StatusConflict, "Coach request already pending")
		return
	}

	levelStr := strings.Join(req.Level, ",")
	if err := h.svc.SetCoachLocality(userID, req.Locality, levelStr); err != nil {
		logErr("update user coach locality and level", err)
	}

	requesterName, requesterAvatar, err := h.svc.GetNameAndAvatar(userID)
	if err != nil {
		logErr("fetch requester name for coach request", err)
	}

	adminIDs, err := h.svc.GetAdminIDs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch admins")
		return
	}

	if len(adminIDs) == 0 {
		log.Println("WARNING: No admin users found for coach request notification")
		writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
		return
	}

	meta := map[string]interface{}{
		"requester_id":     userID,
		"requester_name":   requesterName,
		"requester_avatar": requesterAvatar,
		"locality":         req.Locality,
		"level":            req.Level,
	}
	actions := []models.NotificationAction{
		{Key: "approve", Label: "notif_coach_request_approve", Style: "primary"},
		{Key: "reject", Label: "notif_coach_request_reject", Style: "danger"},
	}

	// Interim: call NotificationHandler directly (until Plan C)
	for _, adminID := range adminIDs {
		h.nh.CreateNotification(adminID, "coach_request",
			"notif_coach_request_title", "notif_coach_request_body",
			meta, actions)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
}

func (h *UserHandler) GetCoachRequestStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	status, err := h.svc.GetCoachRequestStatus(userID)
	if err != nil {
		logErr("get coach request status", err)
		writeError(w, http.StatusInternalServerError, "Failed to check status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

const maxAvatarSize = 500 * 1024 // 500KB base64

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	if !strings.HasPrefix(req.Image, "data:image/") {
		writeError(w, http.StatusBadRequest, "image must be a base64 data URI")
		return
	}

	if len(req.Image) > maxAvatarSize {
		writeError(w, http.StatusBadRequest, "image too large (max 500KB)")
		return
	}

	if err := h.svc.UploadAvatar(userID, req.Image); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar updated"})
}

func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.svc.DeleteAvatar(userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar removed"})
}
