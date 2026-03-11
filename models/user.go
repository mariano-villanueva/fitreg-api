package models

import "time"

type User struct {
	ID                  int64     `json:"id"`
	GoogleID            string    `json:"google_id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name                string  `json:"name"`
	Sex                 string  `json:"sex"`
	BirthDate           string  `json:"birth_date"`
	WeightKg            float64 `json:"weight_kg"`
	HeightCm            int     `json:"height_cm"`
	Language            string  `json:"language"`
	OnboardingCompleted bool    `json:"onboarding_completed"`
}

// CalculateAge returns the age in years given a birth date string (YYYY-MM-DD).
// Returns 0 if the date is empty or invalid.
func CalculateAge(birthDate string) int {
	if birthDate == "" {
		return 0
	}
	bd, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return 0
	}
	now := time.Now()
	age := now.Year() - bd.Year()
	if now.YearDay() < bd.YearDay() {
		age--
	}
	return age
}
