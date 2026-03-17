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
	ApproveAsCoach(id int64) error
}

// TemplateRepository handles all template-related database operations.
type TemplateRepository interface {
	Create(coachID int64, req models.CreateTemplateRequest) (int64, error)
	GetByID(id int64) (models.WorkoutTemplate, error)
	List(coachID int64) ([]models.WorkoutTemplate, error)
	Update(id, coachID int64, req models.CreateTemplateRequest) error
	Delete(id, coachID int64) (bool, error)
	GetSegments(templateID int64) ([]models.TemplateSegment, error)
	ReplaceSegments(templateID int64, segs []models.SegmentRequest) error
	GetCoachID(id int64) (int64, error)
}

// CoachProfileRepository handles all coach profile-related database operations.
type CoachProfileRepository interface {
	UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error
	IsCoach(userID int64) (bool, error)
	ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error)
	GetCoachProfile(coachID int64) (models.CoachPublicProfile, error)
	IsStudentOf(coachID, studentID int64) (bool, error)
	CountStudents(coachID int64) (int, error)
	CountVerifiedAchievements(coachID int64) (int, error)
	GetAchievements(coachID int64) ([]models.CoachAchievement, error)
	GetRatings(coachID int64) ([]models.CoachRating, error)
	GetFileUUID(fileID int64) (string, error)
}

// RatingRepository handles all rating-related database operations.
type RatingRepository interface {
	IsStudentOf(coachID, studentID int64) (bool, error)
	Upsert(coachID, studentID int64, rating int, comment string) error
	List(coachID int64) ([]models.CoachRating, error)
}

// NotificationRepository handles all notification-related database operations.
type NotificationRepository interface {
	// Create inserts a notification after checking preferences for configurable types.
	Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error
	List(userID int64, limit, offset int) ([]models.Notification, error)
	UnreadCount(userID int64) (int, error)
	MarkRead(notifID, userID int64) (bool, error)
	MarkAllRead(userID int64) error
	GetByID(notifID, userID int64) (models.Notification, error)
	ClearActions(notifID int64) error
	ClearActionsByInvitation(userID, invID int64) error
	ClearCancelledInvitation(receiverID, invID int64) error
	ClearCoachRequestActions(requesterID int64) error
	GetPreferences(userID int64) (models.NotificationPreferences, error)
	UpsertPreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error
}

// InvitationRepository handles invitation-related database operations.
type InvitationRepository interface {
	// Used by NotificationService
	GetStatus(id int64) (status string, err error)
	AcceptTx(invitationID, userID int64) (coachID, studentID, senderID int64, err error)
	Reject(invitationID int64) (senderID int64, err error)

	// CRUD used by InvitationService
	FindReceiverByID(receiverID int64) (isCoach, coachPublic bool, err error)
	FindReceiverByEmail(email string) (receiverID int64, isCoach, coachPublic bool, err error)
	IsSenderCoach(senderID int64) (bool, error)
	CountPending(userID, otherID int64) (int, error)
	CountActiveRelationship(userID, otherID int64) (int, error)
	CountStudentActiveCoaches(studentID int64) (int, error)
	Create(senderID, receiverID int64, invType, message string) (invID int64, err error)
	GetByID(id int64) (models.Invitation, error)
	List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error)
	Cancel(invID int64) error
	IsAdmin(userID int64) (bool, error)
}

// AchievementRepository handles all coach achievement database operations.
type AchievementRepository interface {
	List(coachID int64) ([]models.CoachAchievement, error)
	Create(coachID int64, req models.CreateAchievementRequest) (int64, error)
	GetForEdit(achID, coachID int64) (isVerified bool, rejectionReason string, err error)
	Update(achID, coachID int64, req models.UpdateAchievementRequest) error
	Delete(achID, coachID int64) (bool, error)
	SetVisibility(achID, coachID int64, isPublic bool) (bool, error)
	IsCoach(userID int64) (bool, error)
	GetAdminIDs() ([]int64, error)
	// GetFileUUID resolves an image_file_id to its download UUID.
	GetFileUUID(fileID int64) (string, error)
}
