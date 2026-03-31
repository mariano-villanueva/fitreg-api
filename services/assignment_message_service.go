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

func (s *AssignmentMessageService) ListMessages(workoutID, userID int64) ([]models.AssignmentMessage, error) {
	coachID, studentID, _, _, err := s.repo.GetParticipants(workoutID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID != coachID && userID != studentID {
		return nil, ErrForbidden
	}
	return s.repo.List(workoutID)
}

func (s *AssignmentMessageService) SendMessage(workoutID, senderID int64, body string) (models.AssignmentMessage, error) {
	coachID, studentID, status, title, err := s.repo.GetParticipants(workoutID)
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

	msg, err := s.repo.Create(workoutID, senderID, body)
	if err != nil {
		return models.AssignmentMessage{}, err
	}

	recipientID := coachID
	if senderID == coachID {
		recipientID = studentID
	}
	meta := map[string]interface{}{
		"assigned_workout_id": workoutID,
		"workout_title":       title,
		"sender_id":           senderID,
		"sender_name":         msg.SenderName,
	}
	_ = s.notifSvc.Create(recipientID, "assignment_message",
		"notif_assignment_message_title", "notif_assignment_message_body", meta, nil)

	return msg, nil
}

func (s *AssignmentMessageService) MarkRead(workoutID, userID int64) error {
	coachID, studentID, _, _, err := s.repo.GetParticipants(workoutID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if userID != coachID && userID != studentID {
		return ErrForbidden
	}
	return s.repo.MarkRead(workoutID, userID)
}

func (s *AssignmentMessageService) GetWorkoutDetail(workoutID, userID int64) (models.Workout, error) {
	aw, err := s.repo.GetWorkoutDetail(workoutID, userID)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if aw.ImageFileID != nil {
		if uuid, err := s.repo.GetFileUUID(*aw.ImageFileID); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return aw, nil
}
