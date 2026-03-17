package services

import (
	"database/sql"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type WorkoutService struct {
	repo repository.WorkoutRepository
}

func NewWorkoutService(repo repository.WorkoutRepository) *WorkoutService {
	return &WorkoutService{repo: repo}
}

func (s *WorkoutService) List(userID int64) ([]models.Workout, error) {
	return s.repo.List(userID)
}

func (s *WorkoutService) GetByID(id, userID int64) (models.Workout, error) {
	if !s.repo.ExistsByOwner(id, userID) {
		return models.Workout{}, sql.ErrNoRows
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return wo, err
	}
	segs, err := s.repo.GetSegments(id)
	if err == nil {
		wo.Segments = segs
	}
	return wo, nil
}

func (s *WorkoutService) Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
	id, err := s.repo.Create(userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
		return models.Workout{}, err
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return models.Workout{}, err
	}
	segs, _ := s.repo.GetSegments(id)
	wo.Segments = segs
	return wo, nil
}

func (s *WorkoutService) Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
	found, err := s.repo.Update(id, userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if !found {
		return models.Workout{}, sql.ErrNoRows
	}
	if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
		return models.Workout{}, err
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return wo, err
	}
	segs, _ := s.repo.GetSegments(id)
	wo.Segments = segs
	return wo, nil
}

func (s *WorkoutService) Delete(id, userID int64) error {
	found, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if !found {
		return sql.ErrNoRows
	}
	return nil
}
