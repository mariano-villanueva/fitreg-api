package models

import "encoding/json"

type WorkoutTemplate struct {
	ID             int64             `json:"id"`
	CoachID        int64             `json:"coach_id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Type           string            `json:"type"`
	Notes          string            `json:"notes"`
	ExpectedFields json.RawMessage   `json:"expected_fields"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	Segments       []TemplateSegment `json:"segments"`
}

type TemplateSegment struct {
	ID            int64   `json:"id"`
	TemplateID    int64   `json:"template_id"`
	OrderIndex    int     `json:"order_index"`
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

type CreateTemplateRequest struct {
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Type           string           `json:"type"`
	Notes          string           `json:"notes"`
	ExpectedFields []string         `json:"expected_fields"`
	Segments       []SegmentRequest `json:"segments"`
}
