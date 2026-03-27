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
	// CanAccess returns true if userID is allowed to download the file identified by uuid.
	// Access is granted to: the file owner, admins, coaches of the associated assigned workout,
	// and any authenticated user for publicly visible achievement images.
	CanAccess(uuid string, userID int64) (bool, error)
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

// WeeklyTemplateRepository handles CRUD and assignment for weekly workout templates.
type WeeklyTemplateRepository interface {
	Create(coachID int64, req models.CreateWeeklyTemplateRequest) (int64, error)
	GetByID(id int64) (models.WeeklyTemplate, error)
	List(coachID int64) ([]models.WeeklyTemplate, error)
	UpdateMeta(id, coachID int64, req models.UpdateWeeklyTemplateRequest) error
	Delete(id, coachID int64) (bool, error)
	PutDays(templateID int64, days []models.WeeklyTemplateDayRequest) error
	// Assign checks for conflicts and creates assigned_workouts in one transaction.
	// Returns assigned IDs on success, conflicting dates (YYYY-MM-DD) on 409.
	Assign(templateID, coachID int64, req models.AssignWeeklyTemplateRequest) ([]int64, []string, error)
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

	// Token-based invite flow
	CreateForUnknown(senderID int64, invType, message, receiverEmail, inviteToken string) (invID int64, err error)
	FindByToken(token string) (models.Invitation, error)
	RedeemToken(token string, userID int64) error
	FindPendingByEmail(email string) ([]models.Invitation, error)
	SetReceiver(invID, userID int64) error
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

// CoachRepository handles coach-student relationships and assigned workouts.
type CoachRepository interface {
	IsCoach(userID int64) (bool, error)
	IsAdmin(userID int64) (bool, error)
	IsStudentOf(coachID, studentID int64) (bool, error)
	GetStudents(coachID int64) ([]models.CoachStudentInfo, error)
	GetRelationship(csID int64) (coachID, studentID int64, status string, err error)
	EndRelationship(csID int64) error
	GetStudentWorkouts(studentID int64) ([]models.Workout, error)
	ListAssignedWorkouts(coachID int64, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error)
	CreateAssignedWorkout(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error)
	GetAssignedWorkout(awID, coachID int64) (models.AssignedWorkout, error)
	UpdateAssignedWorkout(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error)
	GetAssignedWorkoutStatus(awID, coachID int64) (string, error)
	DeleteAssignedWorkout(awID, coachID int64) error
	GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error)
	UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) (coachID int64, workoutTitle string, err error)
	GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error)
	GetUserName(id int64) (string, error)
	FetchSegments(awID int64) []models.WorkoutSegment
	GetFileUUID(fileID int64) (string, error)
	GetWeeklyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}

// AssignmentMessageRepository handles assignment message and assigned workout detail operations.
type AssignmentMessageRepository interface {
	GetParticipants(awID int64) (coachID, studentID int64, status, title string, err error)
	List(awID int64) ([]models.AssignmentMessage, error)
	Create(awID, senderID int64, body string) (models.AssignmentMessage, error)
	MarkRead(awID, userID int64) error
	GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error)
	// FetchSegments returns segments for an assigned workout.
	FetchSegments(awID int64) []models.WorkoutSegment
	// GetFileUUID resolves a file_id to its download UUID.
	GetFileUUID(fileID int64) (string, error)
}

// AdminRepository handles admin-only database operations.
type AdminRepository interface {
	IsAdmin(userID int64) (bool, error)
	GetStats() (totalUsers, totalCoaches, totalRatings, pendingAchievements int, err error)
	ListUsers(search, role, sortCol, sortOrder string, limit, offset int) (users []models.AdminUser, total int, err error)
	UpdateUserRoles(targetID int64, isCoach, isAdmin *bool) error
	ListPendingAchievements() ([]models.AdminPendingAchievement, error)
	VerifyAchievement(achID, adminID int64) (coachID int64, eventName string, err error)
	RejectAchievement(achID int64, reason string) (coachID int64, eventName string, err error)
	GetFileUUID(fileID int64) (string, error)
}
