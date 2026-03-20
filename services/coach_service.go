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

func (s *CoachService) ListAssignedWorkouts(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
	return s.repo.ListAssignedWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
}

func (s *CoachService) CreateAssignedWorkout(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if !isCoach {
		return models.AssignedWorkout{}, ErrNotCoach
	}
	ok, err := s.repo.IsStudentOf(coachID, req.StudentID)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if !ok {
		return models.AssignedWorkout{}, ErrForbidden
	}
	aw, err := s.repo.CreateAssignedWorkout(coachID, req)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	coachName, _, _ := s.userRepo.GetNameAndAvatar(coachID)
	meta := map[string]interface{}{
		"workout_id":    aw.ID,
		"workout_title": req.Title,
		"coach_name":    coachName,
	}
	_ = s.notifSvc.Create(req.StudentID, "workout_assigned", "notif_workout_assigned_title", "notif_workout_assigned_body", meta, nil)
	return aw, nil
}

func (s *CoachService) GetAssignedWorkout(awID, coachID int64) (models.AssignedWorkout, error) {
	aw, err := s.repo.GetAssignedWorkout(awID, coachID)
	if err == sql.ErrNoRows {
		return models.AssignedWorkout{}, ErrNotFound
	}
	return aw, err
}

func (s *CoachService) UpdateAssignedWorkout(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
	status, err := s.repo.GetAssignedWorkoutStatus(awID, coachID)
	if err == sql.ErrNoRows {
		return models.AssignedWorkout{}, ErrNotFound
	}
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if status != "pending" {
		return models.AssignedWorkout{}, ErrWorkoutFinished
	}
	aw, err := s.repo.UpdateAssignedWorkout(awID, coachID, req)
	if err == sql.ErrNoRows {
		return models.AssignedWorkout{}, ErrNotFound
	}
	return aw, err
}

func (s *CoachService) DeleteAssignedWorkout(awID, coachID int64) error {
	status, err := s.repo.GetAssignedWorkoutStatus(awID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return ErrWorkoutFinished
	}
	err = s.repo.DeleteAssignedWorkout(awID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

func (s *CoachService) GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
	return s.repo.GetMyAssignedWorkouts(studentID, startDate, endDate)
}

func (s *CoachService) UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) error {
	coachID, workoutTitle, err := s.repo.UpdateAssignedWorkoutStatus(awID, studentID, req)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if errors.Is(err, repository.ErrStatusConflict) {
		return ErrWorkoutFinished
	}
	if err != nil {
		return err
	}
	if req.Status == "completed" || req.Status == "skipped" {
		studentName, _, _ := s.userRepo.GetNameAndAvatar(studentID)
		notifType := "workout_completed"
		title, body := "notif_workout_completed_title", "notif_workout_completed_body"
		if req.Status == "skipped" {
			notifType = "workout_skipped"
			title, body = "notif_workout_skipped_title", "notif_workout_skipped_body"
		}
		meta := map[string]interface{}{
			"workout_id":    awID,
			"workout_title": workoutTitle,
			"student_name":  studentName,
		}
		_ = s.notifSvc.Create(coachID, notifType, title, body, meta, nil)
	}
	return nil
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
