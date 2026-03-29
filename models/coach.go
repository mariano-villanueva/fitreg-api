package models

import "time"

type CoachStudent struct {
	ID           int64      `json:"id"`
	CoachID      int64      `json:"coach_id"`
	StudentID    int64      `json:"student_id"`
	InvitationID int64      `json:"invitation_id,omitempty"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type WorkoutSegment struct {
	ID            int64   `json:"id"`
	WorkoutID     int64   `json:"workout_id"`
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

type CoachAchievement struct {
	ID              int64     `json:"id"`
	CoachID         int64     `json:"coach_id"`
	EventName       string    `json:"event_name"`
	EventDate       string    `json:"event_date"`
	DistanceKm      float64   `json:"distance_km"`
	ResultTime      string    `json:"result_time"`
	Position        int       `json:"position"`
	ExtraInfo       string    `json:"extra_info"`
	ImageFileID     *int64    `json:"image_file_id"`
	ImageURL        string    `json:"image_url,omitempty"`
	IsPublic        bool      `json:"is_public"`
	IsVerified      bool      `json:"is_verified"`
	RejectionReason string    `json:"rejection_reason"`
	VerifiedBy      int64     `json:"verified_by,omitempty"`
	VerifiedAt      string    `json:"verified_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateAchievementRequest struct {
	EventName   string  `json:"event_name"`
	EventDate   string  `json:"event_date"`
	DistanceKm  float64 `json:"distance_km"`
	ResultTime  string  `json:"result_time"`
	Position    int     `json:"position"`
	ExtraInfo   string  `json:"extra_info"`
	ImageFileID *int64  `json:"image_file_id"`
}

type UpdateAchievementRequest struct {
	EventName   string  `json:"event_name"`
	EventDate   string  `json:"event_date"`
	DistanceKm  float64 `json:"distance_km"`
	ResultTime  string  `json:"result_time"`
	Position    int     `json:"position"`
	ExtraInfo   string  `json:"extra_info"`
	ImageFileID *int64  `json:"image_file_id"`
}

type CoachRating struct {
	ID          int64     `json:"id"`
	CoachID     int64     `json:"coach_id"`
	StudentID   int64     `json:"student_id"`
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment"`
	StudentName string    `json:"student_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpsertRatingRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type UpdateCoachProfileRequest struct {
	CoachDescription string `json:"coach_description"`
	CoachPublic      bool   `json:"coach_public"`
}

type CoachPublicProfile struct {
	ID                       int64              `json:"id"`
	Name                     string             `json:"name"`
	AvatarURL                string             `json:"avatar_url"`
	CoachDescription         string             `json:"coach_description"`
	AvgRating                float64            `json:"avg_rating"`
	RatingCount              int                `json:"rating_count"`
	StudentCount             int                `json:"student_count"`
	VerifiedAchievementCount int                `json:"verified_achievement_count"`
	IsMyCoach                bool               `json:"is_my_coach"`
	Achievements             []CoachAchievement `json:"achievements"`
	Ratings                  []CoachRating      `json:"ratings"`
}

type CoachListItem struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	AvatarURL        string  `json:"avatar_url"`
	CoachDescription string  `json:"coach_description"`
	CoachLocality    string  `json:"coach_locality"`
	CoachLevel       string  `json:"coach_level"`
	AvgRating        float64 `json:"avg_rating"`
	RatingCount      int     `json:"rating_count"`
	VerifiedCount    int     `json:"verified_achievements"`
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

type CoachStudentInfo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type DailySummaryWorkout struct {
	ID              int64            `json:"id"`
	Title           string           `json:"title"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Description     string           `json:"description"`
	Notes           string           `json:"notes"`
	Status          string           `json:"status"`
	ResultTimeSec   *int             `json:"result_time_seconds"`
	ResultDistKm    *float64         `json:"result_distance_km"`
	ResultHR        *int             `json:"result_heart_rate"`
	ResultFeeling   *int             `json:"result_feeling"`
	DueDate         string           `json:"due_date"`
	Segments        []WorkoutSegment `json:"segments"`
}

type DailySummaryItem struct {
	StudentID     int64                `json:"student_id"`
	StudentName   string               `json:"student_name"`
	StudentAvatar *string              `json:"student_avatar"`
	Workout       *DailySummaryWorkout `json:"workout"`
}

// WeeklyLoadEntry represents the training load for a single week.
type WeeklyLoadEntry struct {
	WeekStart           string  `json:"week_start"` // YYYY-MM-DD (Monday)
	PlannedKm           float64 `json:"planned_km"`
	ActualKm            float64 `json:"actual_km"`
	PlannedSeconds      int     `json:"planned_seconds"`
	ActualSeconds       int     `json:"actual_seconds"`
	SessionsPlanned     int     `json:"sessions_planned"`
	SessionsCompleted   int     `json:"sessions_completed"`
	SessionsSkipped     int     `json:"sessions_skipped"`
	HasPersonalWorkouts bool    `json:"has_personal_workouts"`
}
