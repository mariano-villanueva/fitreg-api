package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/apperr"
)

type AuthHandler struct {
	svc AuthServicer
}

func NewAuthHandler(svc AuthServicer) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type GoogleLoginRequest struct {
	Credential string `json:"credential"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required")
		return
	}

	resp, err := h.svc.GoogleLogin(req.Credential)
	if err != nil {
		writeAppError(w, apperr.New(http.StatusUnauthorized, "AuthHandler.GoogleLogin", apperr.AUTH_001, "Authentication failed", err))
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
