package repository

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/fitreg/api/models"
)

type templateRepository struct {
	db *sql.DB
}

// NewTemplateRepository constructs a TemplateRepository backed by MySQL.
func NewTemplateRepository(db *sql.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Create(coachID int64, req models.CreateTemplateRequest) (int64, error) {
	var expectedFieldsJSON []byte
	var err error
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			log.Printf("ERROR marshal expected fields: %v", err)
		}
	}

	result, err := r.db.Exec(`
		INSERT INTO workout_templates (coach_id, title, description, type, notes, expected_fields)
		VALUES (?, ?, ?, ?, ?, ?)
	`, coachID, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := r.ReplaceSegments(id, req.Segments); err != nil {
		return id, err
	}
	return id, nil
}

func (r *templateRepository) GetByID(id int64) (models.WorkoutTemplate, error) {
	var t models.WorkoutTemplate
	var description, typ, notes, expectedFields sql.NullString
	err := r.db.QueryRow(`
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

	segs, err := r.GetSegments(id)
	if err != nil {
		log.Printf("ERROR fetch segments for template %d: %v", id, err)
		segs = []models.TemplateSegment{}
	}
	t.Segments = segs
	return t, nil
}

func (r *templateRepository) List(coachID int64) ([]models.WorkoutTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, title, description, type, notes, expected_fields, created_at, updated_at
		FROM workout_templates
		WHERE coach_id = ?
		ORDER BY created_at DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := []models.WorkoutTemplate{}
	for rows.Next() {
		var t models.WorkoutTemplate
		var description, typ, notes, expectedFields sql.NullString
		if err := rows.Scan(&t.ID, &t.CoachID, &t.Title, &description, &typ, &notes, &expectedFields, &t.CreatedAt, &t.UpdatedAt); err != nil {
			log.Printf("ERROR scan template row: %v", err)
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
		segs, err := r.GetSegments(templates[i].ID)
		if err != nil {
			log.Printf("ERROR fetch segments for template %d: %v", templates[i].ID, err)
			segs = []models.TemplateSegment{}
		}
		templates[i].Segments = segs
	}

	return templates, nil
}

func (r *templateRepository) Update(id, coachID int64, req models.CreateTemplateRequest) error {
	var expectedFieldsJSON []byte
	var err error
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			log.Printf("ERROR marshal expected fields for template update: %v", err)
		}
	}

	_, err = r.db.Exec(`
		UPDATE workout_templates SET title = ?, description = ?, type = ?, notes = ?, expected_fields = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON, id, coachID)
	if err != nil {
		return err
	}

	return r.ReplaceSegments(id, req.Segments)
}

func (r *templateRepository) Delete(id, coachID int64) (bool, error) {
	result, err := r.db.Exec("DELETE FROM workout_templates WHERE id = ? AND coach_id = ?", id, coachID)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func (r *templateRepository) GetSegments(templateID int64) ([]models.TemplateSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, template_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM workout_template_segments
		WHERE template_id = ?
		ORDER BY order_index ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.TemplateSegment{}
	for rows.Next() {
		var s models.TemplateSegment
		if err := rows.Scan(&s.ID, &s.TemplateID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, nil
}

func (r *templateRepository) ReplaceSegments(templateID int64, segs []models.SegmentRequest) error {
	if _, err := r.db.Exec("DELETE FROM workout_template_segments WHERE template_id = ?", templateID); err != nil {
		return err
	}
	for i, seg := range segs {
		if _, err := r.db.Exec(`
			INSERT INTO workout_template_segments
				(template_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, templateID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return nil
}

func (r *templateRepository) GetCoachID(id int64) (int64, error) {
	var coachID int64
	err := r.db.QueryRow("SELECT coach_id FROM workout_templates WHERE id = ?", id).Scan(&coachID)
	return coachID, err
}
