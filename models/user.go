package models

import (
	"database/sql"
	"time"
)

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

// UserRow is the DB scan struct for the 18-column user query.
type UserRow struct {
	ID                  int64
	GoogleID            string
	Email               string
	Name                string
	AvatarURL           sql.NullString
	CustomAvatar        sql.NullString
	Sex                 sql.NullString
	BirthDate           sql.NullString
	WeightKg            sql.NullFloat64
	HeightCm            sql.NullInt64
	Language            sql.NullString
	IsCoach             sql.NullBool
	IsAdmin             sql.NullBool
	CoachDescription    sql.NullString
	CoachPublic         sql.NullBool
	OnboardingCompleted sql.NullBool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// UserProfile is the API response shape for user profile data.
type UserProfile struct {
	ID                  int64     `json:"id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	CustomAvatar        string    `json:"custom_avatar"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	Age                 int       `json:"age"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	HasCoach            bool      `json:"has_coach"`
	CoachID             int64     `json:"coach_id,omitempty"`
	CoachName           string    `json:"coach_name,omitempty"`
	CoachAvatar         string    `json:"coach_avatar,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
