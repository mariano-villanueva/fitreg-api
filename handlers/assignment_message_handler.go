package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type AssignmentMessageHandler struct {
	svc AssignmentMessageServicer
}

func NewAssignmentMessageHandler(svc AssignmentMessageServicer) *AssignmentMessageHandler {
	return &AssignmentMessageHandler{svc: svc}
}

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
	messages, err := h.svc.ListMessages(awID, userID)
	if err != nil {
		handleServiceErr(w, err, "AssignmentMessageHandler.ListMessages", apperr.ASSIGNMENT_MSG_001, "Failed to fetch messages")
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

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
	var req models.CreateAssignmentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	body := strings.TrimSpace(req.Body)
	if utf8.RuneCountInString(body) < 1 || utf8.RuneCountInString(body) > 2000 {
		writeError(w, http.StatusBadRequest, "Message body must be between 1 and 2000 characters")
		return
	}
	msg, err := h.svc.SendMessage(awID, userID, body)
	if err != nil {
		handleServiceErr(w, err, "AssignmentMessageHandler.SendMessage", apperr.ASSIGNMENT_MSG_002, "Failed to send message")
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

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
	if err := h.svc.MarkRead(awID, userID); err != nil {
		handleServiceErr(w, err, "AssignmentMessageHandler.MarkRead", apperr.ASSIGNMENT_MSG_003, "Failed to mark messages as read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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
	aw, err := h.svc.GetWorkoutDetail(awID, userID)
	if err != nil {
		handleServiceErr(w, err, "AssignmentMessageHandler.GetAssignedWorkoutDetail", apperr.ASSIGNMENT_MSG_004, "Failed to fetch assigned workout")
		return
	}
	writeJSON(w, http.StatusOK, aw)
}
