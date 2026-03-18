package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

// WeeklyTemplateHandler handles /api/coach/weekly-templates endpoints.
type WeeklyTemplateHandler struct {
	svc *services.WeeklyTemplateService
}

// NewWeeklyTemplateHandler creates a new WeeklyTemplateHandler.
func NewWeeklyTemplateHandler(svc *services.WeeklyTemplateService) *WeeklyTemplateHandler {
	return &WeeklyTemplateHandler{svc: svc}
}

// List handles GET /api/coach/weekly-templates
func (h *WeeklyTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	templates, err := h.svc.List(userID)
	if errors.Is(err, services.ErrNotCoach) {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch weekly templates")
		return
	}
	writeJSON(w, http.StatusOK, templates)
}

// Create handles POST /api/coach/weekly-templates
func (h *WeeklyTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateWeeklyTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	tmpl, err := h.svc.Create(userID, req)
	if errors.Is(err, services.ErrNotCoach) {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create weekly template")
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

// Get handles GET /api/coach/weekly-templates/{id}
func (h *WeeklyTemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/weekly-templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid weekly template ID")
		return
	}
	tmpl, err := h.svc.Get(id, userID)
	if errors.Is(err, services.ErrWeeklyTemplateNotFound) {
		writeError(w, http.StatusNotFound, "Weekly template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch weekly template")
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// UpdateMeta handles PUT /api/coach/weekly-templates/{id}
func (h *WeeklyTemplateHandler) UpdateMeta(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/weekly-templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid weekly template ID")
		return
	}
	var req models.UpdateWeeklyTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	tmpl, err := h.svc.UpdateMeta(id, userID, req)
	if errors.Is(err, services.ErrWeeklyTemplateNotFound) {
		writeError(w, http.StatusNotFound, "Weekly template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update weekly template")
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// Delete handles DELETE /api/coach/weekly-templates/{id}
func (h *WeeklyTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/weekly-templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid weekly template ID")
		return
	}
	err = h.svc.Delete(id, userID)
	if errors.Is(err, services.ErrWeeklyTemplateNotFound) {
		writeError(w, http.StatusNotFound, "Weekly template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete weekly template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PutDays handles PUT /api/coach/weekly-templates/{id}/days
func (h *WeeklyTemplateHandler) PutDays(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/weekly-templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid weekly template ID")
		return
	}
	var req models.PutDaysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Days == nil {
		req.Days = []models.WeeklyTemplateDayRequest{}
	}
	tmpl, err := h.svc.PutDays(id, userID, req)
	if errors.Is(err, services.ErrWeeklyTemplateNotFound) {
		writeError(w, http.StatusNotFound, "Weekly template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update days")
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// Assign handles POST /api/coach/weekly-templates/{id}/assign
func (h *WeeklyTemplateHandler) Assign(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/weekly-templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid weekly template ID")
		return
	}
	var req models.AssignWeeklyTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.StudentID == 0 {
		writeError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	if req.StartDate == "" {
		writeError(w, http.StatusBadRequest, "start_date is required")
		return
	}

	resp, err := h.svc.Assign(id, userID, req)
	if errors.Is(err, services.ErrWeeklyTemplateNotFound) {
		writeError(w, http.StatusNotFound, "Weekly template not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		writeError(w, http.StatusForbidden, "Student is not in your roster")
		return
	}
	// Conflict error: extract dates and return 409.
	var conflictErr *services.ConflictError
	if errors.As(err, &conflictErr) {
		writeJSON(w, http.StatusConflict, models.AssignConflictResponse{
			Error:            "conflict",
			ConflictingDates: conflictErr.Dates,
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
