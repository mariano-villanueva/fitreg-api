package services

import (
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// CoachProfileService contains business logic for the coach profile domain.
type CoachProfileService struct {
	repo repository.CoachProfileRepository
}

// NewCoachProfileService constructs a CoachProfileService.
func NewCoachProfileService(repo repository.CoachProfileRepository) *CoachProfileService {
	return &CoachProfileService{repo: repo}
}

// UpdateProfile updates coach description and public flag. Returns ErrNotCoach if not a coach.
func (s *CoachProfileService) UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil || !isCoach {
		return ErrNotCoach
	}
	return s.repo.UpdateProfile(coachID, req)
}

// ListCoaches returns a paginated list of public coaches.
func (s *CoachProfileService) ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
	return s.repo.ListCoaches(search, locality, level, sortBy, limit, offset)
}

// GetCoachProfile returns the full public profile for a coach, including achievements and ratings.
func (s *CoachProfileService) GetCoachProfile(coachID, requestingUserID int64) (models.CoachPublicProfile, error) {
	profile, err := s.repo.GetCoachProfile(coachID)
	if err != nil {
		return profile, err
	}

	// Check if requesting user is a student
	if requestingUserID > 0 {
		isStudent, err := s.repo.IsStudentOf(coachID, requestingUserID)
		if err == nil && isStudent {
			profile.IsMyCoach = true
		}
	}

	// Count students
	studentCount, err := s.repo.CountStudents(coachID)
	if err != nil {
		log.Printf("ERROR count coach students: %v", err)
	}
	profile.StudentCount = studentCount

	// Count verified achievements
	verifiedCount, err := s.repo.CountVerifiedAchievements(coachID)
	if err != nil {
		log.Printf("ERROR count verified achievements: %v", err)
	}
	profile.VerifiedAchievementCount = verifiedCount

	// Fetch achievements
	achievements, err := s.repo.GetAchievements(coachID)
	if err != nil {
		log.Printf("ERROR fetch achievements: %v", err)
		achievements = []models.CoachAchievement{}
	}
	// Resolve image URLs
	for i := range achievements {
		if achievements[i].ImageFileID != nil {
			uuid, err := s.repo.GetFileUUID(*achievements[i].ImageFileID)
			if err == nil {
				achievements[i].ImageURL = "/api/files/" + uuid + "/download"
			}
		}
	}
	if achievements == nil {
		achievements = []models.CoachAchievement{}
	}
	profile.Achievements = achievements

	// Fetch ratings
	ratings, err := s.repo.GetRatings(coachID)
	if err != nil {
		log.Printf("ERROR fetch ratings: %v", err)
		ratings = []models.CoachRating{}
	}
	if ratings == nil {
		ratings = []models.CoachRating{}
	}
	profile.Ratings = ratings

	return profile, nil
}
