package models

import "time"

type Workout struct {
	ID                int64  `json:"id"`
	UserID            int64  `json:"user_id"`
	AssignedWorkoutID *int64 `json:"assigned_workout_id"`
	Date              string `json:"date"`
	DistanceKm      float64   `json:"distance_km"`
	DurationSeconds int       `json:"duration_seconds"`
	AvgPace         string    `json:"avg_pace"`
	Calories        int       `json:"calories"`
	AvgHeartRate    int       `json:"avg_heart_rate"`
	Feeling         *int      `json:"feeling"`
	Type            string    `json:"type"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateWorkoutRequest struct {
	Date            string  `json:"date"`
	DistanceKm      float64 `json:"distance_km"`
	DurationSeconds int     `json:"duration_seconds"`
	AvgPace         string  `json:"avg_pace"`
	Calories        int     `json:"calories"`
	AvgHeartRate    int     `json:"avg_heart_rate"`
	Feeling         *int    `json:"feeling"`
	Type            string  `json:"type"`
	Notes           string  `json:"notes"`
}

type UpdateWorkoutRequest struct {
	Date            string  `json:"date"`
	DistanceKm      float64 `json:"distance_km"`
	DurationSeconds int     `json:"duration_seconds"`
	AvgPace         string  `json:"avg_pace"`
	Calories        int     `json:"calories"`
	AvgHeartRate    int     `json:"avg_heart_rate"`
	Feeling         *int    `json:"feeling"`
	Type            string  `json:"type"`
	Notes           string  `json:"notes"`
}
