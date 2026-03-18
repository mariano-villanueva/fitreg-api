package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type TemplateHandler struct {
	svc *services.TemplateService
}

func NewTemplateHandler(svc *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

// Create handles POST /api/coach/templates
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	tmpl, err := h.svc.Create(userID, req)
	if err != nil {
		handleServiceErr(w, err, "TemplateHandler.Create", "Failed to create template")
		return
	}

	writeJSON(w, http.StatusCreated, tmpl)
}

// List handles GET /api/coach/templates
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	templates, err := h.svc.List(userID)
	if err != nil {
		handleServiceErr(w, err, "TemplateHandler.List", "Failed to fetch templates")
		return
	}

	writeJSON(w, http.StatusOK, templates)
}

// Get handles GET /api/coach/templates/{id}
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	tmpl, err := h.svc.Get(id, userID)
	if err != nil {
		handleServiceErr(w, err, "TemplateHandler.Get", "Failed to fetch template")
		return
	}

	writeJSON(w, http.StatusOK, tmpl)
}

// Update handles PUT /api/coach/templates/{id}
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	tmpl, err := h.svc.Update(id, userID, req)
	if err != nil {
		handleServiceErr(w, err, "TemplateHandler.Update", "Failed to update template")
		return
	}

	writeJSON(w, http.StatusOK, tmpl)
}

// Delete handles DELETE /api/coach/templates/{id}
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	err = h.svc.Delete(id, userID)
	if err != nil {
		handleServiceErr(w, err, "TemplateHandler.Delete", "Failed to delete template")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Template deleted"})
}
