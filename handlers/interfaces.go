package handlers

import (
	"context"
	"io"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

// WorkoutServicer defines the contract for workout business logic used by WorkoutHandler.
type WorkoutServicer interface {
	// Personal
	List(userID int64, startDate, endDate string) ([]models.Workout, error)
	GetByID(id, userID int64) (models.Workout, error)
	Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error)
	Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error)
	Delete(id, userID int64) error
	UpdateStatus(id, userID int64, req models.UpdateWorkoutStatusRequest) error
	// Coach
	GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error)
	CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error)
	ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error)
	GetCoachWorkout(workoutID, coachID int64) (models.Workout, error)
	UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error)
	DeleteCoachWorkout(workoutID, coachID int64) error
}

// UserServicer defines the contract for user business logic used by UserHandler.
type UserServicer interface {
	GetProfile(userID int64) (*models.UserProfile, error)
	UpdateProfile(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error)
	IsCoach(userID int64) (bool, error)
	HasPendingCoachRequest(userID int64) (bool, error)
	SetCoachLocality(userID int64, locality, level string) error
	GetNameAndAvatar(userID int64) (string, string, error)
	GetAdminIDs() ([]int64, error)
	UploadAvatar(userID int64, image string) error
	DeleteAvatar(userID int64) error
	GetCoachRequestStatus(userID int64) (string, error)
}

// NotificationCreator is the narrow interface used by UserHandler (only Create).
type NotificationCreator interface {
	Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error
}

// CoachServicer defines the contract for coach business logic used by CoachHandler.
type CoachServicer interface {
	ListStudents(coachID int64) ([]models.CoachStudentInfo, error)
	EndRelationship(csID, userID int64) error
	GetStudentWorkouts(coachID, studentID int64) ([]models.Workout, error)
	GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error)
	GetStudentLoad(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
	GetMyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}

// TemplateServicer defines the contract for template business logic used by TemplateHandler.
type TemplateServicer interface {
	Create(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error)
	List(coachID int64) ([]models.WorkoutTemplate, error)
	Get(id, coachID int64) (models.WorkoutTemplate, error)
	Update(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error)
	Delete(id, coachID int64) error
}

// InvitationServicer defines the contract for invitation business logic used by InvitationHandler.
type InvitationServicer interface {
	Create(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error)
	List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error)
	GetByID(invID, requestingUserID int64) (models.Invitation, error)
	Respond(invID, userID int64, action string) error
	Cancel(invID, userID int64) error
	Redeem(token string, userID int64) error
}

// NotificationServicer defines the contract for notification business logic used by NotificationHandler.
type NotificationServicer interface {
	List(userID int64, limit, offset int) ([]models.Notification, error)
	UnreadCount(userID int64) (int, error)
	MarkRead(notifID, userID int64) (bool, error)
	MarkAllRead(userID int64) error
	ExecuteAction(notifID, userID int64, action string) error
	GetPreferences(userID int64) (models.NotificationPreferences, error)
	UpdatePreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error
}

// AchievementServicer defines the contract for achievement business logic used by AchievementHandler.
type AchievementServicer interface {
	ListMy(coachID int64) ([]models.CoachAchievement, error)
	Create(coachID int64, req models.CreateAchievementRequest) (int64, error)
	Update(achID, coachID int64, req models.UpdateAchievementRequest) error
	Delete(achID, coachID int64) error
	SetVisibility(achID, coachID int64, isPublic bool) error
}

// AdminServicer defines the contract for admin business logic used by AdminHandler.
type AdminServicer interface {
	GetStats(userID int64) (map[string]int, error)
	ListUsers(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error)
	UpdateUser(callerID, targetID int64, isCoach, isAdmin *bool) error
	ListPendingAchievements(userID int64) ([]models.AdminPendingAchievement, error)
	VerifyAchievement(achID, adminID int64) error
	RejectAchievement(achID, adminID int64, reason string) error
}

// RatingServicer defines the contract for rating business logic used by RatingHandler.
type RatingServicer interface {
	Upsert(coachID, studentID int64, req models.UpsertRatingRequest) error
	List(coachID int64) ([]models.CoachRating, error)
}

// CoachProfileServicer defines the contract for coach profile business logic used by CoachProfileHandler.
type CoachProfileServicer interface {
	UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error
	ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error)
	GetCoachProfile(coachID, requestingUserID int64) (models.CoachPublicProfile, error)
}

// WeeklyTemplateServicer defines the contract for weekly template business logic.
type WeeklyTemplateServicer interface {
	List(coachID int64) ([]models.WeeklyTemplate, error)
	Create(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error)
	Get(id, coachID int64) (models.WeeklyTemplate, error)
	UpdateMeta(id, coachID int64, req models.UpdateWeeklyTemplateRequest) (models.WeeklyTemplate, error)
	Delete(id, coachID int64) error
	PutDays(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error)
	Assign(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error)
}

// AssignmentMessageServicer defines the contract for assignment message business logic.
type AssignmentMessageServicer interface {
	ListMessages(workoutID, userID int64) ([]models.AssignmentMessage, error)
	SendMessage(workoutID, senderID int64, body string) (models.AssignmentMessage, error)
	MarkRead(workoutID, userID int64) error
	GetWorkoutDetail(workoutID, userID int64) (models.Workout, error)
}

// AuthServicer defines the contract for authentication business logic.
type AuthServicer interface {
	GoogleLogin(credential string) (*services.AuthResponse, error)
}

// FileServicer defines the contract for file storage business logic.
type FileServicer interface {
	Upload(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error)
	Download(ctx context.Context, uuid string, userID int64) (string, io.ReadCloser, error)
	Delete(ctx context.Context, uuid string, userID int64) error
}
