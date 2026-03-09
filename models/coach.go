package models

import "time"

type CoachStudent struct {
	ID        int64     `json:"id"`
	CoachID   int64     `json:"coach_id"`
	StudentID int64     `json:"student_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AssignedWorkout struct {
	ID              int64     `json:"id"`
	CoachID         int64     `json:"coach_id"`
	StudentID       int64     `json:"student_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	DistanceKm      float64   `json:"distance_km"`
	DurationSeconds int       `json:"duration_seconds"`
	Notes           string    `json:"notes"`
	Status          string    `json:"status"`
	DueDate         string    `json:"due_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	StudentName     string             `json:"student_name,omitempty"`
	CoachName       string             `json:"coach_name,omitempty"`
	Segments        []WorkoutSegment   `json:"segments"`
}

type AddStudentRequest struct {
	Email string `json:"email"`
}

type CreateAssignedWorkoutRequest struct {
	StudentID       int64   `json:"student_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	DistanceKm      float64 `json:"distance_km"`
	DurationSeconds int     `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}

type UpdateAssignedWorkoutRequest struct {
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}

type UpdateAssignedWorkoutStatusRequest struct {
	Status string `json:"status"`
}

type WorkoutSegment struct {
	ID                int64   `json:"id"`
	AssignedWorkoutID int64   `json:"assigned_workout_id"`
	OrderIndex        int     `json:"order_index"`
	SegmentType       string  `json:"segment_type"`
	Repetitions       int     `json:"repetitions"`
	Value             float64 `json:"value"`
	Unit              string  `json:"unit"`
	Intensity         string  `json:"intensity"`
	WorkValue         float64 `json:"work_value"`
	WorkUnit          string  `json:"work_unit"`
	WorkIntensity     string  `json:"work_intensity"`
	RestValue         float64 `json:"rest_value"`
	RestUnit          string  `json:"rest_unit"`
	RestIntensity     string  `json:"rest_intensity"`
}

type SegmentRequest struct {
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
