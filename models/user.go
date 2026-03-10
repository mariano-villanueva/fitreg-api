package models

import "time"

type User struct {
	ID        int64     `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	Sex       string    `json:"sex"`
	Age       int       `json:"age"`
	WeightKg  float64   `json:"weight_kg"`
	Language  string    `json:"language"`
	IsCoach          bool      `json:"is_coach"`
	IsAdmin          bool      `json:"is_admin"`
	CoachDescription string    `json:"coach_description"`
	CoachPublic      bool      `json:"coach_public"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name     string  `json:"name"`
	Sex      string  `json:"sex"`
	Age      int     `json:"age"`
	WeightKg float64 `json:"weight_kg"`
	Language string  `json:"language"`
	IsCoach  bool    `json:"is_coach"`
}
