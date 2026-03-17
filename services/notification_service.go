package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type NotificationService struct {
	repo     repository.NotificationRepository
	invRepo  repository.InvitationRepository
	userRepo repository.UserRepository
}

func NewNotificationService(
	repo repository.NotificationRepository,
	invRepo repository.InvitationRepository,
	userRepo repository.UserRepository,
) *NotificationService {
	return &NotificationService{repo: repo, invRepo: invRepo, userRepo: userRepo}
}

func (s *NotificationService) Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	return s.repo.Create(userID, notifType, title, body, metadata, actions)
}

func (s *NotificationService) List(userID int64, limit, offset int) ([]models.Notification, error) {
	return s.repo.List(userID, limit, offset)
}

func (s *NotificationService) UnreadCount(userID int64) (int, error) {
	return s.repo.UnreadCount(userID)
}

func (s *NotificationService) MarkRead(notifID, userID int64) (bool, error) {
	return s.repo.MarkRead(notifID, userID)
}

func (s *NotificationService) MarkAllRead(userID int64) error {
	return s.repo.MarkAllRead(userID)
}

func (s *NotificationService) ExecuteAction(notifID, userID int64, action string) error {
	notif, err := s.repo.GetByID(notifID, userID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	// Check actions are available
	if len(notif.Actions) == 0 || string(notif.Actions) == "null" {
		return errors.New("no actions available for this notification")
	}

	// Unmarshal and validate action key
	var actionList []models.NotificationAction
	if err := json.Unmarshal(notif.Actions, &actionList); err != nil {
		return errors.New("invalid actions data")
	}
	validAction := false
	for _, a := range actionList {
		if a.Key == action {
			validAction = true
			break
		}
	}
	if !validAction {
		return errors.New("invalid action")
	}

	switch notif.Type {
	case "invitation_received":
		var meta struct {
			InvitationID int64 `json:"invitation_id"`
		}
		if len(notif.Metadata) > 0 {
			if err := json.Unmarshal(notif.Metadata, &meta); err != nil {
				log.Printf("ERROR unmarshal invitation metadata: %v", err)
			}
		}
		if meta.InvitationID == 0 {
			return errors.New("missing invitation reference")
		}

		status, err := s.invRepo.GetStatus(meta.InvitationID)
		if err != nil {
			return err
		}
		if status != "pending" {
			_ = s.repo.ClearActions(notifID)
			return ErrInvitationNotPending
		}

		switch action {
		case "accept":
			if err := s.acceptInvitation(meta.InvitationID, userID); err != nil {
				return err
			}
		case "reject":
			s.rejectInvitation(meta.InvitationID, userID)
		}

	case "coach_request":
		var meta struct {
			RequesterID   int64  `json:"requester_id"`
			RequesterName string `json:"requester_name"`
		}
		if len(notif.Metadata) > 0 {
			if err := json.Unmarshal(notif.Metadata, &meta); err != nil {
				log.Printf("ERROR unmarshal coach request metadata: %v", err)
			}
		}
		if meta.RequesterID == 0 {
			return errors.New("missing requester reference")
		}

		switch action {
		case "approve":
			s.approveCoachRequest(meta.RequesterID, meta.RequesterName)
		case "reject":
			s.rejectCoachRequest(meta.RequesterID, meta.RequesterName)
		}

		_ = s.repo.ClearCoachRequestActions(meta.RequesterID)

	default:
		return errors.New("unsupported notification type for actions")
	}

	_ = s.repo.ClearActions(notifID)
	return nil
}

func (s *NotificationService) GetPreferences(userID int64) (models.NotificationPreferences, error) {
	return s.repo.GetPreferences(userID)
}

func (s *NotificationService) UpdatePreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error {
	return s.repo.UpsertPreferences(userID, req)
}

// AcceptInvitation is exported for use by InvitationService (Task 2).
func (s *NotificationService) AcceptInvitation(invitationID, userID int64) error {
	return s.acceptInvitation(invitationID, userID)
}

// RejectInvitation is exported for use by InvitationService (Task 2).
func (s *NotificationService) RejectInvitation(invitationID, userID int64) {
	s.rejectInvitation(invitationID, userID)
}

// ClearActionsByInvitation is exported for use by InvitationService (Task 2).
func (s *NotificationService) ClearActionsByInvitation(userID, invID int64) error {
	return s.repo.ClearActionsByInvitation(userID, invID)
}

// ClearCancelledInvitation is exported for use by InvitationService (Task 2).
func (s *NotificationService) ClearCancelledInvitation(receiverID, invID int64) error {
	return s.repo.ClearCancelledInvitation(receiverID, invID)
}

func (s *NotificationService) acceptInvitation(invitationID, userID int64) error {
	_, _, senderID, err := s.invRepo.AcceptTx(invitationID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMaxCoachesReached) {
			return ErrStudentMaxCoaches
		}
		return err
	}

	name, _, _ := s.userRepo.GetNameAndAvatar(userID)
	_ = s.repo.Create(senderID, "invitation_accepted", "notif_invitation_accepted_title", "notif_invitation_accepted_body",
		map[string]interface{}{"invitation_id": invitationID, "user_name": name}, nil)
	return nil
}

func (s *NotificationService) rejectInvitation(invitationID, userID int64) {
	senderID, err := s.invRepo.Reject(invitationID)
	if err != nil {
		log.Printf("ERROR rejecting invitation %d: %v", invitationID, err)
		return
	}

	name, _, _ := s.userRepo.GetNameAndAvatar(userID)
	_ = s.repo.Create(senderID, "invitation_rejected", "notif_invitation_rejected_title", "notif_invitation_rejected_body",
		map[string]interface{}{"invitation_id": invitationID, "user_name": name}, nil)
}

func (s *NotificationService) approveCoachRequest(requesterID int64, requesterName string) {
	if err := s.userRepo.ApproveAsCoach(requesterID); err != nil {
		log.Printf("ERROR approving coach request for user %d: %v", requesterID, err)
	}
	_ = s.repo.Create(requesterID, "coach_request_approved", "notif_coach_request_approved_title", "notif_coach_request_approved_body",
		map[string]interface{}{"requester_name": requesterName}, nil)
}

func (s *NotificationService) rejectCoachRequest(requesterID int64, requesterName string) {
	_ = s.repo.Create(requesterID, "coach_request_rejected", "notif_coach_request_rejected_title", "notif_coach_request_rejected_body",
		map[string]interface{}{"requester_name": requesterName}, nil)
}
