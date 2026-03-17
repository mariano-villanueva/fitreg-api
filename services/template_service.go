package services

import (
	"database/sql"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// TemplateService contains business logic for the template domain.
type TemplateService struct {
	repo     repository.TemplateRepository
	userRepo repository.UserRepository
}

// NewTemplateService constructs a TemplateService.
func NewTemplateService(repo repository.TemplateRepository, userRepo repository.UserRepository) *TemplateService {
	return &TemplateService{repo: repo, userRepo: userRepo}
}

// Create creates a new template. Returns error if user is not a coach.
func (s *TemplateService) Create(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return models.WorkoutTemplate{}, ErrNotCoach
	}

	id, err := s.repo.Create(coachID, req)
	if err != nil {
		return models.WorkoutTemplate{}, err
	}
	return s.repo.GetByID(id)
}

// List returns all templates for a coach. Returns error if user is not a coach.
func (s *TemplateService) List(coachID int64) ([]models.WorkoutTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return nil, ErrNotCoach
	}
	return s.repo.List(coachID)
}

// Get returns a template by ID. Checks ownership.
func (s *TemplateService) Get(id, coachID int64) (models.WorkoutTemplate, error) {
	tmpl, err := s.repo.GetByID(id)
	if err != nil {
		return tmpl, err
	}
	if tmpl.CoachID != coachID {
		return models.WorkoutTemplate{}, sql.ErrNoRows
	}
	return tmpl, nil
}

// Update updates a template. Checks ownership.
func (s *TemplateService) Update(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	ownerID, err := s.repo.GetCoachID(id)
	if err != nil {
		return models.WorkoutTemplate{}, err
	}
	if ownerID != coachID {
		return models.WorkoutTemplate{}, sql.ErrNoRows
	}

	if err := s.repo.Update(id, coachID, req); err != nil {
		return models.WorkoutTemplate{}, err
	}
	return s.repo.GetByID(id)
}

// Delete removes a template. Returns sql.ErrNoRows if not found.
func (s *TemplateService) Delete(id, coachID int64) error {
	found, err := s.repo.Delete(id, coachID)
	if err != nil {
		return err
	}
	if !found {
		return sql.ErrNoRows
	}
	return nil
}
