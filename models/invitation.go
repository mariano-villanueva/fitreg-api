package models

import "time"

const MaxCoachesPerStudent = 1

type Invitation struct {
	ID             int64     `json:"id"`
	Type           string    `json:"type"`
	SenderID       int64     `json:"sender_id"`
	ReceiverID     int64     `json:"receiver_id,omitempty"`
	ReceiverEmail  string    `json:"receiver_email,omitempty"`
	InviteToken    string    `json:"-"`
	Message        string    `json:"message"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SenderName     string    `json:"sender_name,omitempty"`
	SenderAvatar   string    `json:"sender_avatar,omitempty"`
	ReceiverName   string    `json:"receiver_name,omitempty"`
	ReceiverAvatar string    `json:"receiver_avatar,omitempty"`
}

type CreateInvitationRequest struct {
	Type          string `json:"type"`
	ReceiverEmail string `json:"receiver_email"`
	ReceiverID    int64  `json:"receiver_id"`
	Message       string `json:"message"`
}

type RespondInvitationRequest struct {
	Action string `json:"action"`
}

type RedeemInvitationRequest struct {
	Token string `json:"token"`
}
