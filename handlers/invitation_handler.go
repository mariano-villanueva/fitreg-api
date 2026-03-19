package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type InvitationHandler struct {
	svc InvitationServicer
}

func NewInvitationHandler(svc InvitationServicer) *InvitationHandler {
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
		handleServiceErr(w, err, "InvitationHandler.CreateInvitation", apperr.INVITATION_001, "Failed to create invitation")
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
		handleServiceErr(w, err, "InvitationHandler.ListInvitations", apperr.INVITATION_002, "Failed to fetch invitations")
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
		handleServiceErr(w, err, "InvitationHandler.GetInvitation", apperr.INVITATION_003, "Failed to fetch invitation")
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
		handleServiceErr(w, err, "InvitationHandler.RespondInvitation", apperr.INVITATION_004, "Failed to respond to invitation")
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
		handleServiceErr(w, err, "InvitationHandler.CancelInvitation", apperr.INVITATION_005, "Failed to cancel invitation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation cancelled"})
}
