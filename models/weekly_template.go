package models

// WeeklyTemplateSegmentRequest is used when creating/updating weekly template days.
// Identical structure to SegmentRequest in coach.go and template.go.
type WeeklyTemplateSegmentRequest struct {
	SegmentType   string  `json:"segment_type"`
	Repetitions   int     `json:"repetitions"`
	Value         float64 `json:"value"`
	Unit          string  `json:"unit"`
	Intensity     string  `json:"intensity"`
	WorkValue     float64 `json:"work_value"`
	WorkUnit      string  `json:"work_unit"`
	WorkIntensity string  `json:"work_intensity"`
	RestValue     float64 `json:"rest_value"`
	RestUnit      string  `json:"rest_unit"`
	RestIntensity string  `json:"rest_intensity"`
}

// WeeklyTemplateSegment is used in responses.
type WeeklyTemplateSegment struct {
	ID                  int64   `json:"id"`
	WeeklyTemplateDayID int64   `json:"weekly_template_day_id"`
	OrderIndex          int     `json:"order_index"`
	SegmentType         string  `json:"segment_type"`
	Repetitions         int     `json:"repetitions"`
	Value               float64 `json:"value"`
	Unit                string  `json:"unit"`
	Intensity           string  `json:"intensity"`
	WorkValue           float64 `json:"work_value"`
	WorkUnit            string  `json:"work_unit"`
	WorkIntensity       string  `json:"work_intensity"`
	RestValue           float64 `json:"rest_value"`
	RestUnit            string  `json:"rest_unit"`
	RestIntensity       string  `json:"rest_intensity"`
}

// WeeklyTemplateDay represents one day slot in a weekly template (response).
type WeeklyTemplateDay struct {
	ID                 int64                   `json:"id"`
	WeeklyTemplateID   int64                   `json:"weekly_template_id"`
	DayOfWeek          int                     `json:"day_of_week"` // 0=Mon … 6=Sun
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	Type               string                  `json:"type"`
	DistanceKm         float64                 `json:"distance_km"`
	DurationSeconds    int                     `json:"duration_seconds"`
	Notes              string                  `json:"notes"`
	FromTemplateID     *int64                  `json:"from_template_id"`
	Segments           []WeeklyTemplateSegment `json:"segments"`
}

// WeeklyTemplate is the full response object.
type WeeklyTemplate struct {
	ID          int64               `json:"id"`
	CoachID     int64               `json:"coach_id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Days        []WeeklyTemplateDay `json:"days"`
	DayCount    int                 `json:"day_count,omitempty"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// --- Request models ---

// CreateWeeklyTemplateRequest is the body for POST /api/coach/weekly-templates.
type CreateWeeklyTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateWeeklyTemplateRequest is the body for PUT /api/coach/weekly-templates/:id.
type UpdateWeeklyTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// WeeklyTemplateDayRequest is one day entry in PutDaysRequest.
type WeeklyTemplateDayRequest struct {
	DayOfWeek       int                            `json:"day_of_week"`
	Title           string                         `json:"title"`
	Description     string                         `json:"description"`
	Type            string                         `json:"type"`
	DistanceKm      float64                        `json:"distance_km"`
	DurationSeconds int                            `json:"duration_seconds"`
	Notes           string                         `json:"notes"`
	FromTemplateID  *int64                         `json:"from_template_id"`
	Segments        []WeeklyTemplateSegmentRequest `json:"segments"`
}

// PutDaysRequest is the body for PUT /api/coach/weekly-templates/:id/days.
type PutDaysRequest struct {
	Days []WeeklyTemplateDayRequest `json:"days"`
}

// AssignWeeklyTemplateRequest is the body for POST /api/coach/weekly-templates/:id/assign.
type AssignWeeklyTemplateRequest struct {
	StudentID int64  `json:"student_id"`
	StartDate string `json:"start_date"` // "YYYY-MM-DD", must be a Monday
}

// AssignWeeklyTemplateResponse is returned on successful assignment.
type AssignWeeklyTemplateResponse struct {
	AssignedWorkoutIDs []int64 `json:"assigned_workout_ids"`
}

// AssignConflictResponse is returned when one or more dates already have an assigned workout.
type AssignConflictResponse struct {
	Error            string   `json:"error"`
	ConflictingDates []string `json:"conflicting_dates"`
}
