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

type InvitationHandler struct {
	svc *services.InvitationService
}

func NewInvitationHandler(svc *services.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
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

	inv, err := h.svc.Create(userID, req)
	if err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "user_not_found")
		case services.ErrCannotInviteSelf:
			writeError(w, http.StatusBadRequest, "cannot_invite_self")
		case services.ErrNotCoach:
			writeError(w, http.StatusBadRequest, "not_a_coach")
		case services.ErrReceiverNotCoach:
			writeError(w, http.StatusBadRequest, "receiver_not_coach")
		case services.ErrInvitationAlreadyPending:
			writeError(w, http.StatusBadRequest, "invitation_already_pending")
		case services.ErrAlreadyConnected:
			writeError(w, http.StatusBadRequest, "already_connected")
		case services.ErrStudentMaxCoaches:
			writeError(w, http.StatusBadRequest, "student_max_coaches")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to create invitation")
		}
		return
	}

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

	invitations, err := h.svc.List(userID, status, direction, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch invitations")
		return
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

	inv, err := h.svc.GetByID(invID, userID)
	if err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrForbidden:
			writeError(w, http.StatusForbidden, "Access denied")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to fetch invitation")
		}
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

	if err := h.svc.Respond(invID, userID, req.Action); err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrOnlyReceiver:
			writeError(w, http.StatusForbidden, err.Error())
		case services.ErrInvitationNotPending:
			writeError(w, http.StatusConflict, err.Error())
		case services.ErrStudentMaxCoaches:
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to respond to invitation")
		}
		return
	}

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

	if err := h.svc.Cancel(invID, userID); err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrOnlySender:
			writeError(w, http.StatusForbidden, err.Error())
		case services.ErrInvitationNotPending:
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to cancel invitation")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation cancelled"})
}
