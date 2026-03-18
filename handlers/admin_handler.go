package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/services"
)

type AdminHandler struct {
	svc *services.AdminService
}

func NewAdminHandler(svc *services.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	stats, err := h.svc.GetStats(userID)
	if err != nil {
		handleServiceErr(w, err, "AdminHandler.GetStats", "Failed to fetch stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	// Parse query params
	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		search = search[:100]
	}
	role := r.URL.Query().Get("role")
	sortCol := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Defaults and clamping
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Whitelist sort column
	allowedSort := map[string]string{
		"name":       "name",
		"email":      "email",
		"created_at": "created_at",
	}
	if _, ok := allowedSort[sortCol]; !ok {
		sortCol = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	offset := (page - 1) * limit
	users, total, err := h.svc.ListUsers(userID, search, role, sortCol, sortOrder, limit, offset)
	if err != nil {
		handleServiceErr(w, err, "AdminHandler.ListUsers", "Failed to fetch users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	targetID, err := extractID(r.URL.Path, "/api/admin/users/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req struct {
		IsCoach *bool `json:"is_coach"`
		IsAdmin *bool `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.svc.UpdateUser(userID, targetID, req.IsCoach, req.IsAdmin); err != nil {
		handleServiceErr(w, err, "AdminHandler.UpdateUser", "Failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User updated"})
}

func (h *AdminHandler) PendingAchievements(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	achievements, err := h.svc.ListPendingAchievements(userID)
	if err != nil {
		handleServiceErr(w, err, "AdminHandler.PendingAchievements", "Failed to fetch achievements")
		return
	}
	writeJSON(w, http.StatusOK, achievements)
}

func (h *AdminHandler) VerifyAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	path := strings.TrimSuffix(r.URL.Path, "/verify")
	achID, err := extractID(path, "/api/admin/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}

	if err := h.svc.VerifyAchievement(achID, userID); err != nil {
		handleServiceErr(w, err, "AdminHandler.VerifyAchievement", "Failed to verify achievement")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement verified"})
}

func (h *AdminHandler) RejectAchievement(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	path := strings.TrimSuffix(r.URL.Path, "/reject")
	achID, err := extractID(path, "/api/admin/achievements/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid achievement ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for backwards compatibility
		req.Reason = ""
	}

	if err := h.svc.RejectAchievement(achID, userID, req.Reason); err != nil {
		handleServiceErr(w, err, "AdminHandler.RejectAchievement", "Failed to reject achievement")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Achievement rejected"})
}
