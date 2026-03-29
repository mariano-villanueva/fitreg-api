package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fitreg/api/models"
)

type weeklyTemplateRepository struct {
	db *sql.DB
}

// NewWeeklyTemplateRepository creates a new WeeklyTemplateRepository.
func NewWeeklyTemplateRepository(db *sql.DB) WeeklyTemplateRepository {
	return &weeklyTemplateRepository{db: db}
}

func (r *weeklyTemplateRepository) Create(coachID int64, req models.CreateWeeklyTemplateRequest) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO weekly_templates (coach_id, name, description) VALUES (?, ?, ?)`,
		coachID, req.Name, req.Description,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *weeklyTemplateRepository) GetByID(id int64) (models.WeeklyTemplate, error) {
	var wt models.WeeklyTemplate
	var description sql.NullString
	err := r.db.QueryRow(
		`SELECT id, coach_id, name, description, created_at, updated_at FROM weekly_templates WHERE id = ?`, id,
	).Scan(&wt.ID, &wt.CoachID, &wt.Name, &description, &wt.CreatedAt, &wt.UpdatedAt)
	if err != nil {
		return wt, err
	}
	wt.Description = description.String

	days, err := r.getDays(id)
	if err != nil {
		return wt, err
	}
	wt.Days = days
	wt.DayCount = len(days)
	return wt, nil
}

func (r *weeklyTemplateRepository) getDays(templateID int64) ([]models.WeeklyTemplateDay, error) {
	rows, err := r.db.Query(
		`SELECT id, weekly_template_id, day_of_week, title, description, type,
		        distance_km, duration_seconds, notes, from_template_id
		 FROM weekly_template_days WHERE weekly_template_id = ? ORDER BY day_of_week`, templateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var days []models.WeeklyTemplateDay
	for rows.Next() {
		var d models.WeeklyTemplateDay
		var desc, typ, notes sql.NullString
		var distKm sql.NullFloat64
		var durSec sql.NullInt64
		var fromTmplID sql.NullInt64
		if err := rows.Scan(&d.ID, &d.WeeklyTemplateID, &d.DayOfWeek, &d.Title,
			&desc, &typ, &distKm, &durSec, &notes, &fromTmplID); err != nil {
			return nil, err
		}
		d.Description = desc.String
		d.Type = typ.String
		d.Notes = notes.String
		d.DistanceKm = distKm.Float64
		d.DurationSeconds = int(durSec.Int64)
		if fromTmplID.Valid {
			v := fromTmplID.Int64
			d.FromTemplateID = &v
		}
		segs, err := r.getSegments(d.ID)
		if err != nil {
			return nil, err
		}
		d.Segments = segs
		days = append(days, d)
	}
	if days == nil {
		days = []models.WeeklyTemplateDay{}
	}
	return days, rows.Err()
}

func (r *weeklyTemplateRepository) getSegments(dayID int64) ([]models.WeeklyTemplateSegment, error) {
	rows, err := r.db.Query(
		`SELECT id, weekly_template_day_id, order_index, segment_type, repetitions,
		        value, unit, intensity, work_value, work_unit, work_intensity,
		        rest_value, rest_unit, rest_intensity
		 FROM weekly_template_day_segments WHERE weekly_template_day_id = ? ORDER BY order_index`, dayID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segs []models.WeeklyTemplateSegment
	for rows.Next() {
		var s models.WeeklyTemplateSegment
		var val, wv, rv sql.NullFloat64
		var unit, intensity, wu, wi, ru, ri sql.NullString
		if err := rows.Scan(&s.ID, &s.WeeklyTemplateDayID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &val, &unit, &intensity,
			&wv, &wu, &wi, &rv, &ru, &ri); err != nil {
			return nil, err
		}
		s.Value = val.Float64
		s.Unit = unit.String
		s.Intensity = intensity.String
		s.WorkValue = wv.Float64
		s.WorkUnit = wu.String
		s.WorkIntensity = wi.String
		s.RestValue = rv.Float64
		s.RestUnit = ru.String
		s.RestIntensity = ri.String
		segs = append(segs, s)
	}
	if segs == nil {
		segs = []models.WeeklyTemplateSegment{}
	}
	return segs, rows.Err()
}

func (r *weeklyTemplateRepository) List(coachID int64) ([]models.WeeklyTemplate, error) {
	rows, err := r.db.Query(
		`SELECT id, coach_id, name, description, created_at, updated_at FROM weekly_templates
		 WHERE coach_id = ? ORDER BY created_at DESC`, coachID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.WeeklyTemplate
	for rows.Next() {
		var wt models.WeeklyTemplate
		var description sql.NullString
		if err := rows.Scan(&wt.ID, &wt.CoachID, &wt.Name, &description, &wt.CreatedAt, &wt.UpdatedAt); err != nil {
			return nil, err
		}
		wt.Description = description.String

		// Count days without fetching segments for list performance
		var count int
		if err := r.db.QueryRow(
			`SELECT COUNT(*) FROM weekly_template_days WHERE weekly_template_id = ?`, wt.ID,
		).Scan(&count); err != nil {
			return nil, err
		}
		wt.Days = []models.WeeklyTemplateDay{}
		wt.DayCount = count
		templates = append(templates, wt)
	}
	if templates == nil {
		templates = []models.WeeklyTemplate{}
	}
	return templates, rows.Err()
}

func (r *weeklyTemplateRepository) UpdateMeta(id, coachID int64, req models.UpdateWeeklyTemplateRequest) error {
	res, err := r.db.Exec(
		`UPDATE weekly_templates SET name = ?, description = ? WHERE id = ? AND coach_id = ?`,
		req.Name, req.Description, id, coachID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *weeklyTemplateRepository) Delete(id, coachID int64) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM weekly_templates WHERE id = ? AND coach_id = ?`, id, coachID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// PutDays replaces all days (and their segments) for the given template atomically.
func (r *weeklyTemplateRepository) PutDays(templateID int64, days []models.WeeklyTemplateDayRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM weekly_template_days WHERE weekly_template_id = ?`, templateID); err != nil {
		return err
	}

	for _, d := range days {
		res, err := tx.Exec(
			`INSERT INTO weekly_template_days
			 (weekly_template_id, day_of_week, title, description, type, distance_km, duration_seconds, notes, from_template_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			templateID, d.DayOfWeek, d.Title, d.Description, d.Type, d.DistanceKm, d.DurationSeconds, d.Notes, d.FromTemplateID,
		)
		if err != nil {
			return err
		}
		dayID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		for i, seg := range d.Segments {
			if _, err := tx.Exec(
				`INSERT INTO weekly_template_day_segments
				 (weekly_template_day_id, order_index, segment_type, repetitions, value, unit, intensity,
				  work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				dayID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
				seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// Assign creates assigned_workouts for each day with content, starting at startDate (a Monday).
// Returns (assignedIDs, conflictDates, error).
// conflictDates is non-nil when there are conflicts (caller should return 409).
func (r *weeklyTemplateRepository) Assign(templateID, coachID int64, req models.AssignWeeklyTemplateRequest) ([]int64, []string, error) {
	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid start_date: %w", err)
	}

	days, err := r.getDays(templateID)
	if err != nil {
		return nil, nil, err
	}

	// Build the list of (day, dueDate) pairs for days that have content.
	type dayWithDate struct {
		day     models.WeeklyTemplateDay
		dueDate time.Time
	}
	var planned []dayWithDate
	for _, d := range days {
		planned = append(planned, dayWithDate{
			day:     d,
			dueDate: startDate.AddDate(0, 0, d.DayOfWeek), // 0=Mon → same day, 6=Sun → +6 days
		})
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	if req.Force {
		// Delete the entire week for this student so the template overwrites everything.
		weekStart := startDate.Format(time.DateOnly)
		weekEnd := startDate.AddDate(0, 0, 6).Format(time.DateOnly)
		if _, err := tx.Exec(
			`DELETE FROM workouts WHERE user_id = ? AND coach_id = ? AND due_date >= ? AND due_date <= ?`,
			req.StudentID, coachID, weekStart, weekEnd,
		); err != nil {
			return nil, nil, err
		}
	} else {
		// Check for conflicts.
		var conflicting []string
		for _, p := range planned {
			dateStr := p.dueDate.Format(time.DateOnly)
			var exists int
			if err := tx.QueryRow(
				`SELECT COUNT(*) FROM workouts WHERE user_id = ? AND coach_id = ? AND due_date = ?`,
				req.StudentID, coachID, dateStr,
			).Scan(&exists); err != nil {
				return nil, nil, err
			}
			if exists > 0 {
				conflicting = append(conflicting, dateStr)
			}
		}
		if len(conflicting) > 0 {
			return nil, conflicting, nil
		}
	}

	// Insert assigned_workouts and segments.
	var ids []int64
	for _, p := range planned {
		dateStr := p.dueDate.Format(time.DateOnly)
		res, err := tx.Exec(
			`INSERT INTO workouts
			 (coach_id, user_id, title, description, type, distance_km, duration_seconds, notes, due_date, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			coachID, req.StudentID, p.day.Title, p.day.Description, p.day.Type,
			p.day.DistanceKm, p.day.DurationSeconds, p.day.Notes, dateStr,
		)
		if err != nil {
			return nil, nil, err
		}
		awID, err := res.LastInsertId()
		if err != nil {
			return nil, nil, err
		}
		for i, seg := range p.day.Segments {
			if _, err := tx.Exec(
				`INSERT INTO workout_segments
				 (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				  work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				awID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
				seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity,
			); err != nil {
				return nil, nil, err
			}
		}
		ids = append(ids, awID)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return ids, nil, nil
}
