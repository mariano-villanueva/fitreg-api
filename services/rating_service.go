package services

import (
	"errors"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// ErrNotStudent is returned when a user is not a student of the given coach.
var ErrNotStudent = errors.New("you are not a student of this coach")

// ErrInvalidRating is returned when a rating value is out of range.
var ErrInvalidRating = errors.New("rating must be between 1 and 10")

// RatingService contains business logic for the rating domain.
type RatingService struct {
	repo repository.RatingRepository
}

// NewRatingService constructs a RatingService.
func NewRatingService(repo repository.RatingRepository) *RatingService {
	return &RatingService{repo: repo}
}

// Upsert creates or updates a rating. Checks student relationship and validates range.
func (s *RatingService) Upsert(coachID, studentID int64, req models.UpsertRatingRequest) error {
	isStudent, err := s.repo.IsStudentOf(coachID, studentID)
	if err != nil {
		return err
	}
	if !isStudent {
		return errors.New("not a student of this coach")
	}
	if req.Rating < 1 || req.Rating > 10 {
		return ErrInvalidRating
	}
	return s.repo.Upsert(coachID, studentID, req.Rating, req.Comment)
}

// List returns all ratings for a coach.
func (s *RatingService) List(coachID int64) ([]models.CoachRating, error) {
	return s.repo.List(coachID)
}
