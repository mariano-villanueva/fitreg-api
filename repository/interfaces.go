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
