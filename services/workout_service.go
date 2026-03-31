package services

import (
	"database/sql"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type WorkoutService struct {
	repo      repository.WorkoutRepository
	notifSvc  *NotificationService
	userRepo  repository.UserRepository
	coachRepo repository.CoachRepository
}

func NewWorkoutService(repo repository.WorkoutRepository, notifSvc *NotificationService, userRepo repository.UserRepository, coachRepo repository.CoachRepository) *WorkoutService {
	return &WorkoutService{repo: repo, notifSvc: notifSvc, userRepo: userRepo, coachRepo: coachRepo}
}

func withSegments(svc *WorkoutService, wo models.Workout) models.Workout {
	segs, err := svc.repo.GetSegments(wo.ID)
	if err != nil {
		log.Printf("ERROR fetching segments for workout %d: %v", wo.ID, err)
		wo.Segments = []models.WorkoutSegment{}
	} else {
		wo.Segments = segs
	}
	if wo.ImageFileID != nil {
		if uuid, err := svc.repo.GetFileUUID(*wo.ImageFileID); err == nil {
			wo.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return wo
}

// --- Personal workout methods ---

func (s *WorkoutService) List(userID int64, startDate, endDate string) ([]models.Workout, error) {
	workouts, err := s.repo.List(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	for i := range workouts {
		workouts[i] = withSegments(s, workouts[i])
	}
	return workouts, nil
}

func (s *WorkoutService) GetByID(id, userID int64) (models.Workout, error) {
	wo, err := s.repo.GetByID(id)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if wo.UserID != userID && (wo.CoachID == nil || *wo.CoachID != userID) {
		return models.Workout{}, ErrForbidden
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
	id, err := s.repo.Create(userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if len(req.Segments) > 0 {
		if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
			return models.Workout{}, err
		}
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
	found, err := s.repo.Update(id, userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if !found {
		return models.Workout{}, ErrNotFound
	}
	if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
		return models.Workout{}, err
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Delete(id, userID int64) error {
	found, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *WorkoutService) UpdateStatus(id, userID int64, req models.UpdateWorkoutStatusRequest) error {
	coachID, workoutTitle, err := s.repo.UpdateStatus(id, userID, req)
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
		studentName, _, _ := s.userRepo.GetNameAndAvatar(userID)
		notifType := "workout_completed"
		title, body := "notif_workout_completed_title", "notif_workout_completed_body"
		if req.Status == "skipped" {
			notifType = "workout_skipped"
			title, body = "notif_workout_skipped_title", "notif_workout_skipped_body"
		}
		meta := map[string]interface{}{
			"workout_id":    id,
			"workout_title": workoutTitle,
			"student_name":  studentName,
		}
		_ = s.notifSvc.Create(coachID, notifType, title, body, meta, nil)
	}
	return nil
}

// --- Coach workout methods ---

func (s *WorkoutService) GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error) {
	workouts, err := s.repo.GetMyWorkouts(studentID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	for i := range workouts {
		workouts[i] = withSegments(s, workouts[i])
	}
	return workouts, nil
}

func (s *WorkoutService) CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error) {
	isCoach, err := s.coachRepo.IsCoach(coachID)
	if err != nil {
		return models.Workout{}, err
	}
	if !isCoach {
		return models.Workout{}, ErrNotCoach
	}
	ok, err := s.coachRepo.IsStudentOf(coachID, req.StudentID)
	if err != nil {
		return models.Workout{}, err
	}
	if !ok {
		return models.Workout{}, ErrForbidden
	}
	wo, err := s.repo.CreateCoachWorkout(coachID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if len(req.Segments) > 0 {
		if err := s.repo.ReplaceSegments(wo.ID, req.Segments); err != nil {
			return models.Workout{}, err
		}
	}
	coachName, _, _ := s.userRepo.GetNameAndAvatar(coachID)
	meta := map[string]interface{}{
		"workout_id":    wo.ID,
		"workout_title": req.Title,
		"coach_name":    coachName,
	}
	_ = s.notifSvc.Create(req.StudentID, "workout_assigned", "notif_workout_assigned_title", "notif_workout_assigned_body", meta, nil)
	return withSegments(s, wo), nil
}

func (s *WorkoutService) ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error) {
	return s.repo.ListCoachWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
}

func (s *WorkoutService) GetCoachWorkout(workoutID, coachID int64) (models.Workout, error) {
	wo, err := s.repo.GetCoachWorkout(workoutID, coachID)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error) {
	status, err := s.repo.GetWorkoutStatus(workoutID, coachID)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if status != "pending" {
		return models.Workout{}, ErrWorkoutFinished
	}
	wo, err := s.repo.UpdateCoachWorkout(workoutID, coachID, req)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if err := s.repo.ReplaceSegments(workoutID, req.Segments); err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) DeleteCoachWorkout(workoutID, coachID int64) error {
	status, err := s.repo.GetWorkoutStatus(workoutID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return ErrWorkoutFinished
	}
	err = s.repo.DeleteCoachWorkout(workoutID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}
