package services

import (
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// UserService contains business logic for the user domain.
type UserService struct {
	repo repository.UserRepository
}

// NewUserService constructs a UserService.
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GetProfile returns the full user profile with coach info.
func (s *UserService) GetProfile(userID int64) (*models.UserProfile, error) {
	row, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	u := rowToUserProfile(row)
	hasCoach, err := s.repo.HasActiveCoach(userID)
	if err != nil {
		log.Printf("ERROR check has coach for profile: %v", err)
	}
	u.HasCoach = hasCoach
	if hasCoach {
		fillCoachInfo(s.repo, userID, &u)
	}
	return &u, nil
}

// UpdateProfile updates the user and returns the refreshed profile.
func (s *UserService) UpdateProfile(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error) {
	if err := s.repo.UpdateProfile(userID, req); err != nil {
		return nil, err
	}
	row, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	u := rowToUserProfile(row)
	return &u, nil
}

// GetCoachRequestStatus returns "approved", "pending", or "none".
func (s *UserService) GetCoachRequestStatus(userID int64) (string, error) {
	isCoach, err := s.repo.IsCoach(userID)
	if err != nil {
		return "", err
	}
	if isCoach {
		return "approved", nil
	}
	pending, err := s.repo.HasPendingCoachRequest(userID)
	if err != nil {
		return "", err
	}
	if pending {
		return "pending", nil
	}
	return "none", nil
}

// IsCoach checks if a user is a coach.
func (s *UserService) IsCoach(userID int64) (bool, error) {
	return s.repo.IsCoach(userID)
}

// HasPendingCoachRequest checks if there is already a pending coach request.
func (s *UserService) HasPendingCoachRequest(userID int64) (bool, error) {
	return s.repo.HasPendingCoachRequest(userID)
}

// SetCoachLocality updates the user's coach locality and level.
func (s *UserService) SetCoachLocality(userID int64, locality, level string) error {
	return s.repo.SetCoachLocality(userID, locality, level)
}

// GetNameAndAvatar returns a user's name and avatar.
func (s *UserService) GetNameAndAvatar(userID int64) (string, string, error) {
	return s.repo.GetNameAndAvatar(userID)
}

// GetAdminIDs returns all admin user IDs.
func (s *UserService) GetAdminIDs() ([]int64, error) {
	return s.repo.GetAdminIDs()
}

// UploadAvatar sets the custom avatar for a user.
func (s *UserService) UploadAvatar(userID int64, image string) error {
	return s.repo.UploadAvatar(userID, image)
}

// DeleteAvatar removes the custom avatar for a user.
func (s *UserService) DeleteAvatar(userID int64) error {
	return s.repo.DeleteAvatar(userID)
}
