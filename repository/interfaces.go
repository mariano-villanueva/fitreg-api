package repository

import "github.com/fitreg/api/models"

// WorkoutRepository handles all workout-related database operations.
type WorkoutRepository interface {
	List(userID int64) ([]models.Workout, error)
	GetByID(id int64) (models.Workout, error)
	ExistsByOwner(id, userID int64) bool
	Create(userID int64, req models.CreateWorkoutRequest) (int64, error)
	Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error)
	Delete(id, userID int64) (bool, error)
	GetSegments(workoutID int64) ([]models.WorkoutSegment, error)
	ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error
}

// FileRepository handles all file-related database operations.
type FileRepository interface {
	Create(uuid string, userID int64, name, contentType string, size int64, storageKey string) (models.File, error)
	GetByUUID(uuid string) (models.File, error)
	GetOwnerAndKey(uuid string) (userID int64, storageKey string, err error)
	Delete(uuid string) error
}

// UserRepository handles all user-related database operations.
// Used by both AuthService and UserService.
type UserRepository interface {
	FindByGoogleID(googleID string) (models.UserRow, error)
	Create(googleID, email, name, avatarURL string) (int64, error)
	UpdateOnLogin(googleID, email, name, picture string) error
	GetByID(id int64) (models.UserRow, error)
	HasActiveCoach(id int64) (bool, error)
	GetActiveCoach(studentID int64) (coachID int64, name, avatar string, found bool)
	UpdateProfile(id int64, req models.UpdateProfileRequest) error
	IsCoach(id int64) (bool, error)
	HasPendingCoachRequest(id int64) (bool, error)
	SetCoachLocality(id int64, locality, level string) error
	GetNameAndAvatar(id int64) (name, avatar string, err error)
	GetAdminIDs() ([]int64, error)
	UploadAvatar(id int64, image string) error
	DeleteAvatar(id int64) error
}
