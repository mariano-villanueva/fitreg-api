package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// ErrWeeklyTemplateNotFound is returned when the template does not exist or belongs to another coach.
var ErrWeeklyTemplateNotFound = errors.New("weekly template not found")

// ConflictError carries the conflicting dates for the 409 response.
type ConflictError struct {
	Dates []string
}

func (e *ConflictError) Error() string { return "assignment conflict" }

// WeeklyTemplateService encapsulates business logic for weekly templates.
type WeeklyTemplateService struct {
	repo      repository.WeeklyTemplateRepository
	coachRepo repository.CoachRepository
	userRepo  repository.UserRepository
}

// NewWeeklyTemplateService creates a new WeeklyTemplateService.
func NewWeeklyTemplateService(
	repo repository.WeeklyTemplateRepository,
	coachRepo repository.CoachRepository,
	userRepo repository.UserRepository,
) *WeeklyTemplateService {
	return &WeeklyTemplateService{repo: repo, coachRepo: coachRepo, userRepo: userRepo}
}

func (s *WeeklyTemplateService) List(coachID int64) ([]models.WeeklyTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return nil, ErrNotCoach
	}
	return s.repo.List(coachID)
}

func (s *WeeklyTemplateService) Create(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return models.WeeklyTemplate{}, ErrNotCoach
	}
	if req.Name == "" {
		return models.WeeklyTemplate{}, errors.New("name is required")
	}
	id, err := s.repo.Create(coachID, req)
	if err != nil {
		return models.WeeklyTemplate{}, err
	}
	return s.repo.GetByID(id)
}

func (s *WeeklyTemplateService) Get(id, coachID int64) (models.WeeklyTemplate, error) {
	wt, err := s.repo.GetByID(id)
	if err == sql.ErrNoRows {
		return wt, ErrWeeklyTemplateNotFound
	}
	if err != nil {
		return wt, err
	}
	if wt.CoachID != coachID {
		return models.WeeklyTemplate{}, ErrWeeklyTemplateNotFound
	}
	return wt, nil
}

func (s *WeeklyTemplateService) UpdateMeta(id, coachID int64, req models.UpdateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
	if _, err := s.Get(id, coachID); err != nil {
		return models.WeeklyTemplate{}, err
	}
	if req.Name == "" {
		return models.WeeklyTemplate{}, errors.New("name is required")
	}
	if err := s.repo.UpdateMeta(id, coachID, req); err != nil {
		if err == sql.ErrNoRows {
			return models.WeeklyTemplate{}, ErrWeeklyTemplateNotFound
		}
		return models.WeeklyTemplate{}, err
	}
	return s.repo.GetByID(id)
}

func (s *WeeklyTemplateService) Delete(id, coachID int64) error {
	if _, err := s.Get(id, coachID); err != nil {
		return err
	}
	found, err := s.repo.Delete(id, coachID)
	if err != nil {
		return err
	}
	if !found {
		return ErrWeeklyTemplateNotFound
	}
	return nil
}

func (s *WeeklyTemplateService) PutDays(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error) {
	if _, err := s.Get(id, coachID); err != nil {
		return models.WeeklyTemplate{}, err
	}

	// Validate day_of_week range and uniqueness.
	seen := map[int]bool{}
	for _, d := range req.Days {
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return models.WeeklyTemplate{}, errors.New("day_of_week must be between 0 and 6")
		}
		if seen[d.DayOfWeek] {
			return models.WeeklyTemplate{}, errors.New("duplicate day_of_week in request")
		}
		seen[d.DayOfWeek] = true
		if d.Title == "" {
			return models.WeeklyTemplate{}, errors.New("each day must have a title")
		}
	}

	if err := s.repo.PutDays(id, req.Days); err != nil {
		return models.WeeklyTemplate{}, err
	}
	return s.repo.GetByID(id)
}

func (s *WeeklyTemplateService) Assign(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error) {
	// Ownership check.
	if _, err := s.Get(id, coachID); err != nil {
		return models.AssignWeeklyTemplateResponse{}, err
	}

	// start_date must be a Monday.
	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		return models.AssignWeeklyTemplateResponse{}, errors.New("invalid start_date format, expected YYYY-MM-DD")
	}
	if startDate.Weekday() != time.Monday {
		return models.AssignWeeklyTemplateResponse{}, errors.New("start_date must be a Monday")
	}

	// Student must be in coach roster.
	isStudent, err := s.coachRepo.IsStudentOf(coachID, req.StudentID)
	if err != nil {
		return models.AssignWeeklyTemplateResponse{}, err
	}
	if !isStudent {
		return models.AssignWeeklyTemplateResponse{}, ErrForbidden
	}

	ids, conflicts, err := s.repo.Assign(id, coachID, req)
	if err != nil {
		return models.AssignWeeklyTemplateResponse{}, err
	}
	if len(conflicts) > 0 {
		return models.AssignWeeklyTemplateResponse{}, &ConflictError{Dates: conflicts}
	}
	if ids == nil {
		ids = []int64{}
	}
	return models.AssignWeeklyTemplateResponse{AssignedWorkoutIDs: ids}, nil
}
