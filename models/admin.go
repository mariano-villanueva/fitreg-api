package models

type AdminUser struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	IsCoach   bool   `json:"is_coach"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt string `json:"created_at"`
}

type AdminPendingAchievement struct {
	ID          int64   `json:"id"`
	CoachID     int64   `json:"coach_id"`
	EventName   string  `json:"event_name"`
	EventDate   string  `json:"event_date"`
	DistanceKm  float64 `json:"distance_km"`
	ResultTime  string  `json:"result_time"`
	Position    int     `json:"position"`
	ExtraInfo   string  `json:"extra_info"`
	ImageFileID *int64  `json:"image_file_id"`
	ImageURL    string  `json:"image_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
	CoachName   string  `json:"coach_name"`
}
