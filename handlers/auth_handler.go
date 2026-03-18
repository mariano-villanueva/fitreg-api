package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/fitreg/api/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
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
		log.Printf("ERROR GoogleLogin: %v", err)
		writeError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
