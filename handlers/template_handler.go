package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type TemplateHandler struct {
	DB *sql.DB
}

func NewTemplateHandler(db *sql.DB) *TemplateHandler {
	return &TemplateHandler{DB: db}
}

// Create handles POST /api/coach/templates
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var isCoach bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach); err != nil || !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
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

	var expectedFieldsJSON []byte
	var err error
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			logErr("marshal expected fields", err)
		}
	}

	result, err := h.DB.Exec(`
		INSERT INTO workout_templates (coach_id, title, description, type, notes, expected_fields)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON)
	if err != nil {
		log.Printf("ERROR creating template: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create template")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for template", err)
	}

	for i, seg := range req.Segments {
		_, err := h.DB.Exec(`
			INSERT INTO workout_template_segments
				(template_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity)
		if err != nil {
			log.Printf("ERROR inserting template segment %d: %v", i, err)
			writeError(w, http.StatusInternalServerError, "Failed to create template segment")
			return
		}
	}

	tmpl, err := h.fetchTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch created template")
		return
	}
	tmpl.Segments = h.fetchTemplateSegments(id)

	writeJSON(w, http.StatusCreated, tmpl)
}

func (h *TemplateHandler) fetchTemplate(id int64) (models.WorkoutTemplate, error) {
	var t models.WorkoutTemplate
	var description, typ, notes, expectedFields sql.NullString
	err := h.DB.QueryRow(`
		SELECT id, coach_id, title, description, type, notes, expected_fields, created_at, updated_at
		FROM workout_templates WHERE id = ?
	`, id).Scan(&t.ID, &t.CoachID, &t.Title, &description, &typ, &notes, &expectedFields, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if description.Valid {
		t.Description = description.String
	}
	if typ.Valid {
		t.Type = typ.String
	}
	if notes.Valid {
		t.Notes = notes.String
	}
	if expectedFields.Valid {
		t.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	return t, nil
}

func (h *TemplateHandler) fetchTemplateSegments(templateID int64) []models.TemplateSegment {
	rows, err := h.DB.Query(`
		SELECT id, template_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM workout_template_segments
		WHERE template_id = ?
		ORDER BY order_index ASC
	`, templateID)
	if err != nil {
		logErr("fetch template segments query", err)
		return []models.TemplateSegment{}
	}
	defer rows.Close()

	segments := []models.TemplateSegment{}
	for rows.Next() {
		var s models.TemplateSegment
		if err := rows.Scan(&s.ID, &s.TemplateID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			logErr("scan template segment row", err)
			continue
		}
		segments = append(segments, s)
	}
	return segments
}

// List handles GET /api/coach/templates
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var isCoach bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach); err != nil || !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, coach_id, title, description, type, notes, expected_fields, created_at, updated_at
		FROM workout_templates
		WHERE coach_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch templates")
		return
	}
	defer rows.Close()

	templates := []models.WorkoutTemplate{}
	for rows.Next() {
		var t models.WorkoutTemplate
		var description, typ, notes, expectedFields sql.NullString
		if err := rows.Scan(&t.ID, &t.CoachID, &t.Title, &description, &typ, &notes, &expectedFields, &t.CreatedAt, &t.UpdatedAt); err != nil {
			logErr("scan template row", err)
			continue
		}
		if description.Valid {
			t.Description = description.String
		}
		if typ.Valid {
			t.Type = typ.String
		}
		if notes.Valid {
			t.Notes = notes.String
		}
		if expectedFields.Valid {
			t.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		templates = append(templates, t)
	}

	for i := range templates {
		templates[i].Segments = h.fetchTemplateSegments(templates[i].ID)
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

	tmpl, err := h.fetchTemplate(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch template")
		return
	}
	if tmpl.CoachID != userID {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	tmpl.Segments = h.fetchTemplateSegments(id)

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

	var coachID int64
	err = h.DB.QueryRow("SELECT coach_id FROM workout_templates WHERE id = ?", id).Scan(&coachID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch template")
		return
	}
	if coachID != userID {
		writeError(w, http.StatusNotFound, "Template not found")
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

	var expectedFieldsJSON []byte
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			logErr("marshal expected fields for template update", err)
		}
	}

	_, err = h.DB.Exec(`
		UPDATE workout_templates SET title = ?, description = ?, type = ?, notes = ?, expected_fields = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON, id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update template")
		return
	}

	// Replace segments: delete old, insert new
	if _, err := h.DB.Exec("DELETE FROM workout_template_segments WHERE template_id = ?", id); err != nil {
		logErr("delete old template segments", err)
	}
	for i, seg := range req.Segments {
		if _, err := h.DB.Exec(`
			INSERT INTO workout_template_segments
				(template_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			logErr("insert updated template segment", err)
		}
	}

	tmpl, err := h.fetchTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch updated template")
		return
	}
	tmpl.Segments = h.fetchTemplateSegments(id)

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

	result, err := h.DB.Exec("DELETE FROM workout_templates WHERE id = ? AND coach_id = ?", id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete template")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Template deleted"})
}
