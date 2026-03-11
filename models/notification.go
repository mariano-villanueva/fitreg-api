package models

import (
	"encoding/json"
	"time"
)

type NotificationAction struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Style string `json:"style"`
}

type Notification struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Metadata  json.RawMessage `json:"metadata"`
	Actions   json.RawMessage `json:"actions"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}

type NotificationPreferences struct {
	ID                        int64 `json:"id"`
	UserID                    int64 `json:"user_id"`
	WorkoutAssigned           bool  `json:"workout_assigned"`
	WorkoutCompletedOrSkipped bool  `json:"workout_completed_or_skipped"`
}

type UpdateNotificationPreferencesRequest struct {
	WorkoutAssigned           bool `json:"workout_assigned"`
	WorkoutCompletedOrSkipped bool `json:"workout_completed_or_skipped"`
}

type NotificationActionRequest struct {
	Action string `json:"action"`
}
