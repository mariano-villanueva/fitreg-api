package models

import "time"

type AssignmentMessage struct {
	ID           int64     `json:"id"`
	WorkoutID    int64     `json:"workout_id"`
	SenderID     int64     `json:"sender_id"`
	SenderName   string    `json:"sender_name"`
	SenderAvatar string    `json:"sender_avatar"`
	Body         string    `json:"body"`
	IsRead       bool      `json:"is_read"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateAssignmentMessageRequest struct {
	Body string `json:"body"`
}
