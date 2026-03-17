package services

import (
	"database/sql"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

var (
	// ErrCannotInviteSelf is returned when a user tries to invite themselves.
	ErrCannotInviteSelf = errors.New("cannot_invite_self")
	// ErrReceiverNotCoach is returned when a student_request targets a non-public coach.
	ErrReceiverNotCoach = errors.New("receiver_not_coach")
	// ErrInvitationAlreadyPending is returned when a pending invitation already exists between the two users.
	ErrInvitationAlreadyPending = errors.New("invitation_already_pending")
	// ErrAlreadyConnected is returned when an active coach-student relationship already exists.
	ErrAlreadyConnected = errors.New("already_connected")
	// ErrOnlyReceiver is returned when a non-receiver tries to respond to an invitation.
	ErrOnlyReceiver = errors.New("only the receiver can respond")
	// ErrOnlySender is returned when a non-sender tries to cancel an invitation.
	ErrOnlySender = errors.New("only the sender can cancel")
)

type InvitationService struct {
	repo     repository.InvitationRepository
	notifSvc *NotificationService
	userRepo repository.UserRepository
}

func NewInvitationService(
	repo repository.InvitationRepository,
	notifSvc *NotificationService,
	userRepo repository.UserRepository,
) *InvitationService {
	return &InvitationService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *InvitationService) Create(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error) {
	// 1. Resolve receiver
	var receiverID int64
	var receiverIsCoach, receiverCoachPublic bool

	if req.ReceiverID > 0 {
		receiverID = req.ReceiverID
		isCoach, coachPublic, err := s.repo.FindReceiverByID(req.ReceiverID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.Invitation{}, ErrNotFound
			}
			return models.Invitation{}, err
		}
		receiverIsCoach = isCoach
		receiverCoachPublic = coachPublic
	} else {
		rID, isCoach, coachPublic, err := s.repo.FindReceiverByEmail(req.ReceiverEmail)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.Invitation{}, ErrNotFound
			}
			return models.Invitation{}, err
		}
		receiverID = rID
		receiverIsCoach = isCoach
		receiverCoachPublic = coachPublic
	}

	// 2. No self-invitation
	if senderID == receiverID {
		return models.Invitation{}, ErrCannotInviteSelf
	}

	// 3. Type-specific validation
	if req.Type == "coach_invite" {
		isCoach, err := s.repo.IsSenderCoach(senderID)
		if err != nil {
			return models.Invitation{}, err
		}
		if !isCoach {
			return models.Invitation{}, ErrNotCoach
		}
	} else if req.Type == "student_request" {
		if !receiverIsCoach || !receiverCoachPublic {
			return models.Invitation{}, ErrReceiverNotCoach
		}
	} else {
		return models.Invitation{}, errors.New("invalid invitation type")
	}

	// 6. Check no pending invitation exists
	count, err := s.repo.CountPending(senderID, receiverID)
	if err != nil {
		return models.Invitation{}, err
	}
	if count > 0 {
		return models.Invitation{}, ErrInvitationAlreadyPending
	}

	// 7. Check no active relationship exists
	count, err = s.repo.CountActiveRelationship(senderID, receiverID)
	if err != nil {
		return models.Invitation{}, err
	}
	if count > 0 {
		return models.Invitation{}, ErrAlreadyConnected
	}

	// 8. Determine studentID
	var studentID int64
	if req.Type == "coach_invite" {
		studentID = receiverID
	} else {
		studentID = senderID
	}

	// 9. Check student active coaches limit
	coachCount, err := s.repo.CountStudentActiveCoaches(studentID)
	if err != nil {
		return models.Invitation{}, err
	}
	if coachCount >= models.MaxCoachesPerStudent {
		return models.Invitation{}, ErrStudentMaxCoaches
	}

	// 10. Create invitation
	invID, err := s.repo.Create(senderID, receiverID, req.Type, req.Message)
	if err != nil {
		return models.Invitation{}, err
	}

	// 11. Notify receiver
	senderName, senderAvatar, _ := s.userRepo.GetNameAndAvatar(senderID)
	meta := map[string]interface{}{
		"invitation_id": invID,
		"sender_id":     senderID,
		"sender_name":   senderName,
		"sender_avatar": senderAvatar,
	}
	actions := []models.NotificationAction{
		{Key: "accept", Label: "invitation_accept", Style: "primary"},
		{Key: "reject", Label: "invitation_reject", Style: "danger"},
	}
	var title, body string
	if req.Type == "coach_invite" {
		title = "notif_coach_invite_title"
		body = "notif_coach_invite_body"
	} else {
		title = "notif_student_request_title"
		body = "notif_student_request_body"
	}
	if err := s.notifSvc.Create(receiverID, "invitation_received", title, body, meta, actions); err != nil {
		log.Printf("ERROR creating invitation notification: %v", err)
	}

	// 12. Return created invitation
	return s.repo.GetByID(invID)
}

func (s *InvitationService) List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
	return s.repo.List(userID, status, direction, limit, offset)
}

func (s *InvitationService) GetByID(invID, requestingUserID int64) (models.Invitation, error) {
	inv, err := s.repo.GetByID(invID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Invitation{}, ErrNotFound
		}
		return models.Invitation{}, err
	}

	isAdmin, _ := s.repo.IsAdmin(requestingUserID)
	if inv.SenderID != requestingUserID && inv.ReceiverID != requestingUserID && !isAdmin {
		return models.Invitation{}, ErrForbidden
	}

	return inv, nil
}

func (s *InvitationService) Respond(invID, userID int64, action string) error {
	// 1. Validate action
	if action != "accepted" && action != "rejected" {
		return errors.New("action must be 'accepted' or 'rejected'")
	}

	// 2. Fetch invitation
	inv, err := s.repo.GetByID(invID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	// 3. Check admin
	isAdmin, _ := s.repo.IsAdmin(userID)

	// 4. Only receiver (or admin) can respond
	if inv.ReceiverID != userID && !isAdmin {
		return ErrOnlyReceiver
	}

	// 5. Must be pending
	if inv.Status != "pending" {
		return ErrInvitationNotPending
	}

	// 6/7. Accept or reject
	if action == "accepted" {
		if err := s.notifSvc.AcceptInvitation(invID, userID); err != nil {
			return err
		}
	} else {
		s.notifSvc.RejectInvitation(invID, userID)
	}

	// 8. Clear actions on related notification
	if err := s.notifSvc.ClearActionsByInvitation(userID, invID); err != nil {
		log.Printf("ERROR clearing invitation notification actions: %v", err)
	}

	return nil
}

func (s *InvitationService) Cancel(invID, userID int64) error {
	// 1. Fetch invitation
	inv, err := s.repo.GetByID(invID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	// 2. Check admin
	isAdmin, _ := s.repo.IsAdmin(userID)

	// 3. Only sender (or admin) can cancel
	if inv.SenderID != userID && !isAdmin {
		return ErrOnlySender
	}

	// 4. Must be pending
	if inv.Status != "pending" {
		return ErrInvitationNotPending
	}

	// 5. Cancel in DB
	if err := s.repo.Cancel(invID); err != nil {
		return err
	}

	// 6. Clear notification for receiver
	if err := s.notifSvc.ClearCancelledInvitation(inv.ReceiverID, invID); err != nil {
		log.Printf("ERROR clearing cancelled invitation notification: %v", err)
	}

	return nil
}
