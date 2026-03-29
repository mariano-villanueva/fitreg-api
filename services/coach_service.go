package services

import (
	"database/sql"
	"errors"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)


type CoachService struct {
	repo     repository.CoachRepository
	notifSvc *NotificationService
	userRepo repository.UserRepository
}

func NewCoachService(repo repository.CoachRepository, notifSvc *NotificationService, userRepo repository.UserRepository) *CoachService {
	return &CoachService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *CoachService) ListStudents(coachID int64) ([]models.CoachStudentInfo, error) {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil {
		return nil, err
	}
	if !isCoach {
		return nil, ErrNotCoach
	}
	return s.repo.GetStudents(coachID)
}

func (s *CoachService) EndRelationship(csID, userID int64) error {
	coachID, studentID, status, err := s.repo.GetRelationship(csID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	isAdmin, _ := s.repo.IsAdmin(userID)
	if coachID != userID && studentID != userID && !isAdmin {
		return ErrForbidden
	}
	if status != "active" {
		return errors.New("relationship is not active")
	}
	if err := s.repo.EndRelationship(csID); err != nil {
		return err
	}
	otherID := studentID
	if userID == studentID {
		otherID = coachID
	}
	name, _, _ := s.userRepo.GetNameAndAvatar(userID)
	meta := map[string]interface{}{"user_id": userID, "user_name": name}
	_ = s.notifSvc.Create(otherID, "relationship_ended", "notif_relationship_ended_title", "notif_relationship_ended_body", meta, nil)
	return nil
}

func (s *CoachService) GetStudentWorkouts(coachID, studentID int64) ([]models.Workout, error) {
	ok, err := s.repo.IsStudentOf(coachID, studentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	return s.repo.GetStudentWorkouts(studentID)
}

func (s *CoachService) GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error) {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil {
		return nil, err
	}
	if !isCoach {
		return nil, ErrNotCoach
	}
	return s.repo.GetDailySummary(coachID, date, includeSegments)
}

func (s *CoachService) GetStudentLoad(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	ok, err := s.repo.IsStudentOf(coachID, studentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	return s.repo.GetWeeklyLoad(studentID, weeks)
}

func (s *CoachService) GetMyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	return s.repo.GetWeeklyLoad(studentID, weeks)
}
