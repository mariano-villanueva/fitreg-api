package services

import (
	"database/sql"
	"errors"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type AssignmentMessageService struct {
	repo     repository.AssignmentMessageRepository
	notifSvc *NotificationService
}

func NewAssignmentMessageService(repo repository.AssignmentMessageRepository, notifSvc *NotificationService) *AssignmentMessageService {
	return &AssignmentMessageService{repo: repo, notifSvc: notifSvc}
}

func (s *AssignmentMessageService) ListMessages(awID, userID int64) ([]models.AssignmentMessage, error) {
	coachID, studentID, _, _, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID != coachID && userID != studentID {
		return nil, ErrForbidden
	}
	return s.repo.List(awID)
}

func (s *AssignmentMessageService) SendMessage(awID, senderID int64, body string) (models.AssignmentMessage, error) {
	coachID, studentID, status, title, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return models.AssignmentMessage{}, ErrNotFound
	}
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	if senderID != coachID && senderID != studentID {
		return models.AssignmentMessage{}, ErrForbidden
	}
	if status != "pending" {
		return models.AssignmentMessage{}, errors.New("cannot send messages on a non-pending assignment")
	}

	msg, err := s.repo.Create(awID, senderID, body)
	if err != nil {
		return models.AssignmentMessage{}, err
	}

	recipientID := coachID
	if senderID == coachID {
		recipientID = studentID
	}
	meta := map[string]interface{}{
		"assigned_workout_id": awID,
		"workout_title":       title,
		"sender_id":           senderID,
		"sender_name":         msg.SenderName,
	}
	_ = s.notifSvc.Create(recipientID, "assignment_message",
		"notif_assignment_message_title", "notif_assignment_message_body", meta, nil)

	return msg, nil
}

func (s *AssignmentMessageService) MarkRead(awID, userID int64) error {
	coachID, studentID, _, _, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if userID != coachID && userID != studentID {
		return ErrForbidden
	}
	return s.repo.MarkRead(awID, userID)
}

func (s *AssignmentMessageService) GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error) {
	aw, err := s.repo.GetAssignedWorkoutDetail(awID, userID)
	if err == sql.ErrNoRows {
		return models.AssignedWorkout{}, ErrNotFound
	}
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	aw.Segments = s.repo.FetchSegments(awID)
	if aw.ImageFileID != nil {
		if uuid, err := s.repo.GetFileUUID(*aw.ImageFileID); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return aw, nil
}
