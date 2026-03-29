package models

import (
	"encoding/json"
	"time"
)

type Workout struct {
	ID                 int64            `json:"id"`
	UserID             int64            `json:"user_id"`
	CoachID            *int64           `json:"coach_id"`
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	Type               string           `json:"type"`
	Notes              string           `json:"notes"`
	DueDate            string           `json:"due_date"`
	DistanceKm         float64          `json:"distance_km"`
	DurationSeconds    int              `json:"duration_seconds"`
	ExpectedFields     json.RawMessage  `json:"expected_fields"`
	ResultDistanceKm   *float64         `json:"result_distance_km"`
	ResultTimeSeconds  *int             `json:"result_time_seconds"`
	ResultHeartRate    *int             `json:"result_heart_rate"`
	ResultFeeling      *int             `json:"result_feeling"`
	AvgPace            string           `json:"avg_pace"`
	Calories           int              `json:"calories"`
	ImageFileID        *int64           `json:"image_file_id"`
	Status             string           `json:"status"`
	Segments           []WorkoutSegment `json:"segments"`
	ImageURL           string           `json:"image_url,omitempty"`
	UnreadMessageCount int              `json:"unread_message_count"`
	CoachName          string           `json:"coach_name,omitempty"`
	UserName           string           `json:"user_name,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// CreateWorkoutRequest is used by athletes to log a personal workout.
type CreateWorkoutRequest struct {
	DueDate           string           `json:"due_date"`
	DistanceKm        float64          `json:"distance_km"`
	DurationSeconds   int              `json:"duration_seconds"`
	AvgPace           string           `json:"avg_pace"`
	Calories          int              `json:"calories"`
	ResultDistanceKm  *float64         `json:"result_distance_km"`
	ResultTimeSeconds *int             `json:"result_time_seconds"`
	ResultHeartRate   *int             `json:"result_heart_rate"`
	ResultFeeling     *int             `json:"result_feeling"`
	Type              string           `json:"type"`
	Notes             string           `json:"notes"`
	Segments          []SegmentRequest `json:"segments"`
}

// UpdateWorkoutRequest is used by athletes to edit a personal workout.
type UpdateWorkoutRequest struct {
	DueDate           string           `json:"due_date"`
	DistanceKm        float64          `json:"distance_km"`
	DurationSeconds   int              `json:"duration_seconds"`
	AvgPace           string           `json:"avg_pace"`
	Calories          int              `json:"calories"`
	ResultDistanceKm  *float64         `json:"result_distance_km"`
	ResultTimeSeconds *int             `json:"result_time_seconds"`
	ResultHeartRate   *int             `json:"result_heart_rate"`
	ResultFeeling     *int             `json:"result_feeling"`
	Type              string           `json:"type"`
	Notes             string           `json:"notes"`
	Segments          []SegmentRequest `json:"segments"`
}

// UpdateWorkoutStatusRequest is used by athletes to complete/skip a coach-assigned workout.
type UpdateWorkoutStatusRequest struct {
	Status            string   `json:"status"`
	ResultTimeSeconds *int     `json:"result_time_seconds"`
	ResultDistanceKm  *float64 `json:"result_distance_km"`
	ResultHeartRate   *int     `json:"result_heart_rate"`
	ResultFeeling     *int     `json:"result_feeling"`
	ImageFileID       *int64   `json:"image_file_id"`
}

// CreateCoachWorkoutRequest is used by coaches to assign a workout to a student.
type CreateCoachWorkoutRequest struct {
	StudentID       int64            `json:"student_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	ExpectedFields  []string         `json:"expected_fields"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}

// UpdateCoachWorkoutRequest is used by coaches to edit a coach-assigned workout.
type UpdateCoachWorkoutRequest struct {
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	ExpectedFields  []string         `json:"expected_fields"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}
