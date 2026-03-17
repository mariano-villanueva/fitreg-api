# API Refactor Plan C: Domains 8-13 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the 6 remaining monolithic handlers (notification, invitation, achievement, assignment_message, coach, admin) to the `handlers→services→repository` architecture, and clean up the `UserHandler`'s interim `*NotificationHandler` dependency.

**Architecture:** Same pattern as Plans A and B. Each domain gets a repository (SQL only, implements interface in `repository/interfaces.go`), a service (business logic, no `*sql.DB`), and a slimmed handler (HTTP parsing + service call). Notable difference: `NotificationService` is the foundation layer depended on by every other new service — it must be done first. `InvitationRepository` is partially defined in Task 1 to break the accept/reject circular dependency, then extended in Task 2.

**Tech Stack:** Go 1.24, stdlib HTTP, `*sql.DB` MySQL, UberFX (`go.uber.org/fx`)

---

## Conventions

- Module path: `github.com/fitreg/api`
- Repository interfaces all live in `repository/interfaces.go` (single growing file)
- Service sentinel errors live in `services/errors.go` (already exists)
- `truncateDate(s string) string` helper already exists in both `services/user_projection.go` (for services) and `handlers/helpers.go` (for handlers) — use the in-package copy
- `handlers/helpers.go` has `extractID`, `truncateDate` — handlers keep using these
- After each task: `go build ./...` must pass; commit
- `main.go` pattern: add `repository.NewXxxRepository, services.NewXxxService` to `fx.Provide`, update handler constructor signature

## Dependency Order

```
NotificationService ← needed by all below
InvitationService   ← needs NotificationService
AchievementService  ← needs NotificationService
AssignmentMessageService ← needs NotificationService
CoachService        ← needs NotificationService
AdminService        ← needs NotificationService
UserService (cleanup) ← replace *NotificationHandler with *NotificationService
```

## `main.go` final state (after all 7 tasks)

```go
fx.Provide(
    config.Load,
    dbprovider.New,
    storage.New,
    // Workout domain
    repository.NewWorkoutRepository,
    services.NewWorkoutService,
    // File domain
    repository.NewFileRepository,
    services.NewFileService,
    // Auth + User domain
    repository.NewUserRepository,
    services.NewAuthService,
    services.NewUserService,
    // Template domain
    repository.NewTemplateRepository,
    services.NewTemplateService,
    // CoachProfile domain
    repository.NewCoachProfileRepository,
    services.NewCoachProfileService,
    // Rating domain
    repository.NewRatingRepository,
    services.NewRatingService,
    // Notification domain (Task 1)
    repository.NewNotificationRepository,
    repository.NewInvitationRepository, // stub in T1, full in T2
    services.NewNotificationService,
    // Invitation domain (Task 2)
    services.NewInvitationService,
    // Achievement domain (Task 3)
    repository.NewAchievementRepository,
    services.NewAchievementService,
    // AssignmentMessage domain (Task 4)
    repository.NewAssignmentMessageRepository,
    services.NewAssignmentMessageService,
    // Coach domain (Task 5)
    repository.NewCoachRepository,
    services.NewCoachService,
    // Admin domain (Task 6)
    repository.NewAdminRepository,
    services.NewAdminService,
    // Handlers
    handlers.NewAuthHandler,
    handlers.NewWorkoutHandler,
    handlers.NewCoachProfileHandler,
    handlers.NewRatingHandler,
    handlers.NewTemplateHandler,
    handlers.NewNotificationHandler,
    handlers.NewUserHandler,
    handlers.NewAchievementHandler,
    handlers.NewAssignmentMessageHandler,
    handlers.NewInvitationHandler,
    handlers.NewAdminHandler,
    handlers.NewCoachHandler,
    handlers.NewFileHandler,
    router.New,
),
```

---

## Chunk 1: Notification Domain (Foundation)

### Task 1: Notification Domain + InvitationRepository stub + UserRepository.ApproveAsCoach

This is the most important task. `NotificationService` will be injected into every subsequent service. It also extracts the `acceptInvitation`/`rejectInvitation`/`approveCoachRequest`/`rejectCoachRequest` private logic that currently lives in `NotificationHandler`.

**Files:**
- Modify: `repository/interfaces.go` — add `NotificationRepository`, stub `InvitationRepository`, add `ApproveAsCoach` to `UserRepository`
- Create: `repository/notification_repository.go`
- Create: `repository/invitation_repository.go` (partial — only accept/reject/status methods)
- Modify: `repository/user_repository.go` — add `ApproveAsCoach`
- Create: `services/notification_service.go`
- Modify: `services/errors.go` — add `ErrInvitationNotPending`, `ErrStudentMaxCoaches`
- Modify: `handlers/notification_handler.go` — replace `DB *sql.DB` with `svc *services.NotificationService`
- Modify: `main.go` — add new providers

#### Step 1.1: Add interfaces to `repository/interfaces.go`

- [ ] Add to `repository/interfaces.go` (append after existing interfaces):

```go
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
	// ClearActionsByInvitation nullifies actions on invitation_received notifications for a user+invitation.
	ClearActionsByInvitation(userID, invID int64) error
	// ClearCancelledInvitation nullifies actions and marks the notification as read/cancelled.
	ClearCancelledInvitation(receiverID, invID int64) error
	// ClearCoachRequestActions nullifies actions on all coach_request notifications for a requester.
	ClearCoachRequestActions(requesterID int64) error
	GetPreferences(userID int64) (models.NotificationPreferences, error)
	UpsertPreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error
}

// InvitationRepository handles invitation-related database operations.
// Methods for CRUD are added in Task 2; this stub provides what NotificationService needs.
type InvitationRepository interface {
	GetStatus(id int64) (status string, err error)
	// AcceptTx runs an atomic transaction: creates coach_student record, updates invitation to accepted.
	// Returns coachID, studentID, senderID for post-tx notification.
	AcceptTx(invitationID, userID int64) (coachID, studentID, senderID int64, err error)
	// Reject updates the invitation to rejected and returns the sender ID.
	Reject(invitationID int64) (senderID int64, err error)
}
```

- [ ] Add `ApproveAsCoach(id int64) error` to the existing `UserRepository` interface in `repository/interfaces.go`:

```go
// In the UserRepository interface, add:
ApproveAsCoach(id int64) error
```

#### Step 1.2: Create `repository/notification_repository.go`

- [ ] Create `repository/notification_repository.go`:

```go
package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/fitreg/api/models"
)

type notificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	// Check preferences for configurable types
	if notifType == "workout_assigned" || notifType == "workout_completed" || notifType == "workout_skipped" || notifType == "assignment_message" {
		var workoutAssigned, workoutCompletedOrSkipped, assignmentMessage bool
		err := r.db.QueryRow(
			"SELECT COALESCE(workout_assigned, TRUE), COALESCE(workout_completed_or_skipped, TRUE), COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?",
			userID,
		).Scan(&workoutAssigned, &workoutCompletedOrSkipped, &assignmentMessage)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			workoutAssigned, workoutCompletedOrSkipped, assignmentMessage = true, true, true
		}
		if notifType == "workout_assigned" && !workoutAssigned {
			return nil
		}
		if (notifType == "workout_completed" || notifType == "workout_skipped") && !workoutCompletedOrSkipped {
			return nil
		}
		if notifType == "assignment_message" && !assignmentMessage {
			return nil
		}
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	var actionsJSON []byte
	if actions != nil {
		actionsJSON, err = json.Marshal(actions)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Exec(
		`INSERT INTO notifications (user_id, type, title, body, metadata, actions) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, title, body, metaJSON, actionsJSON,
	)
	return err
}

func (r *notificationRepository) List(userID int64, limit, offset int) ([]models.Notification, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		var metadata, actions sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &actions, &n.IsRead, &n.CreatedAt); err != nil {
			continue
		}
		if metadata.Valid {
			n.Metadata = json.RawMessage(metadata.String)
		}
		if actions.Valid {
			n.Actions = json.RawMessage(actions.String)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *notificationRepository) UnreadCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = FALSE", userID).Scan(&count)
	return count, err
}

func (r *notificationRepository) MarkRead(notifID, userID int64) (bool, error) {
	result, err := r.db.Exec("UPDATE notifications SET is_read = TRUE WHERE id = ? AND user_id = ?", notifID, userID)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *notificationRepository) MarkAllRead(userID int64) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = TRUE WHERE user_id = ? AND is_read = FALSE", userID)
	return err
}

func (r *notificationRepository) GetByID(notifID, userID int64) (models.Notification, error) {
	var n models.Notification
	var metadata, actions sql.NullString
	err := r.db.QueryRow(`
		SELECT id, user_id, type, title, COALESCE(body, ''), metadata, actions, is_read, created_at
		FROM notifications WHERE id = ? AND user_id = ?
	`, notifID, userID).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &actions, &n.IsRead, &n.CreatedAt)
	if err != nil {
		return models.Notification{}, err
	}
	if metadata.Valid {
		n.Metadata = json.RawMessage(metadata.String)
	}
	if actions.Valid {
		n.Actions = json.RawMessage(actions.String)
	}
	return n, nil
}

func (r *notificationRepository) ClearActions(notifID int64) error {
	_, err := r.db.Exec("UPDATE notifications SET actions = NULL WHERE id = ?", notifID)
	return err
}

func (r *notificationRepository) ClearActionsByInvitation(userID, invID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL
		WHERE type = 'invitation_received' AND user_id = ? AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, userID, invID)
	return err
}

func (r *notificationRepository) ClearCancelledInvitation(receiverID, invID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL, body = 'invitation_cancelled', is_read = TRUE
		WHERE type = 'invitation_received' AND user_id = ? AND JSON_EXTRACT(metadata, '$.invitation_id') = ?
	`, receiverID, invID)
	return err
}

func (r *notificationRepository) ClearCoachRequestActions(requesterID int64) error {
	_, err := r.db.Exec(`
		UPDATE notifications SET actions = NULL
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, requesterID)
	return err
}

func (r *notificationRepository) GetPreferences(userID int64) (models.NotificationPreferences, error) {
	var prefs models.NotificationPreferences
	err := r.db.QueryRow(
		"SELECT id, user_id, workout_assigned, workout_completed_or_skipped, COALESCE(assignment_message, TRUE) FROM notification_preferences WHERE user_id = ?",
		userID,
	).Scan(&prefs.ID, &prefs.UserID, &prefs.WorkoutAssigned, &prefs.WorkoutCompletedOrSkipped, &prefs.AssignmentMessage)
	if err == sql.ErrNoRows {
		return models.NotificationPreferences{UserID: userID, WorkoutAssigned: true, WorkoutCompletedOrSkipped: true, AssignmentMessage: true}, nil
	}
	return prefs, err
}

func (r *notificationRepository) UpsertPreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error {
	_, err := r.db.Exec(`
		INSERT INTO notification_preferences (user_id, workout_assigned, workout_completed_or_skipped, assignment_message)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE workout_assigned = VALUES(workout_assigned),
			workout_completed_or_skipped = VALUES(workout_completed_or_skipped),
			assignment_message = VALUES(assignment_message)
	`, userID, req.WorkoutAssigned, req.WorkoutCompletedOrSkipped, req.AssignmentMessage)
	return err
}
```

#### Step 1.3: Create `repository/invitation_repository.go` (partial)

- [ ] Create `repository/invitation_repository.go` with only the methods needed by `NotificationService`:

```go
package repository

import (
	"database/sql"
	"fmt"

	"github.com/fitreg/api/models"
)

type invitationRepository struct {
	db *sql.DB
}

// NewInvitationRepository constructs an InvitationRepository.
// Extended with full CRUD in Task 2.
func NewInvitationRepository(db *sql.DB) InvitationRepository {
	return &invitationRepository{db: db}
}

func (r *invitationRepository) GetStatus(id int64) (string, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM invitations WHERE id = ?", id).Scan(&status)
	return status, err
}

// AcceptTx creates the coach_student relationship and marks the invitation accepted in a transaction.
// Returns (coachID, studentID, senderID) for use in post-transaction notifications.
func (r *invitationRepository) AcceptTx(invitationID, userID int64) (int64, int64, int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	var invType string
	var senderID, receiverID int64
	err = tx.QueryRow(
		"SELECT type, sender_id, receiver_id FROM invitations WHERE id = ? AND status = 'pending' FOR UPDATE",
		invitationID,
	).Scan(&invType, &senderID, &receiverID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invitation not found or already resolved")
	}

	var coachID, studentID int64
	if invType == "coach_invite" {
		coachID, studentID = senderID, receiverID
	} else {
		coachID, studentID = receiverID, senderID
	}

	var activeCount int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active' FOR UPDATE",
		studentID,
	).Scan(&activeCount); err != nil {
		return 0, 0, 0, err
	}
	if activeCount >= models.MaxCoachesPerStudent {
		return 0, 0, 0, fmt.Errorf("student has reached the maximum number of coaches (%d)", models.MaxCoachesPerStudent)
	}

	if _, err = tx.Exec(
		"INSERT INTO coach_students (coach_id, student_id, invitation_id, status, started_at) VALUES (?, ?, ?, 'active', NOW())",
		coachID, studentID, invitationID,
	); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create relationship")
	}
	if _, err = tx.Exec(
		"UPDATE invitations SET status = 'accepted', updated_at = NOW() WHERE id = ?",
		invitationID,
	); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to update invitation")
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, 0, err
	}
	return coachID, studentID, senderID, nil
}

// Reject updates the invitation to rejected and returns the sender ID.
func (r *invitationRepository) Reject(invitationID int64) (int64, error) {
	if _, err := r.db.Exec(
		"UPDATE invitations SET status = 'rejected', updated_at = NOW() WHERE id = ?",
		invitationID,
	); err != nil {
		return 0, err
	}
	var senderID int64
	if err := r.db.QueryRow("SELECT sender_id FROM invitations WHERE id = ?", invitationID).Scan(&senderID); err != nil {
		return 0, err
	}
	return senderID, nil
}
```

#### Step 1.4: Add `ApproveAsCoach` to `repository/user_repository.go`

- [ ] In `repository/user_repository.go`, add the following method to the `userRepository` struct:

```go
func (r *userRepository) ApproveAsCoach(id int64) error {
	_, err := r.db.Exec("UPDATE users SET is_coach = TRUE, coach_public = TRUE, updated_at = NOW() WHERE id = ?", id)
	return err
}
```

#### Step 1.5: Add errors to `services/errors.go`

- [ ] Append to `services/errors.go`:

```go
var ErrInvitationNotPending = errors.New("invitation is no longer pending")
var ErrStudentMaxCoaches    = errors.New("student has reached the maximum number of coaches")
```

#### Step 1.6: Create `services/notification_service.go`

- [ ] Create `services/notification_service.go`:

```go
package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// NotificationService contains business logic for notifications.
// It also owns the accept/reject invitation actions and approve/reject coach_request actions
// because these originate from notification ExecuteAction flows.
type NotificationService struct {
	repo     repository.NotificationRepository
	invRepo  repository.InvitationRepository
	userRepo repository.UserRepository
}

func NewNotificationService(
	repo repository.NotificationRepository,
	invRepo repository.InvitationRepository,
	userRepo repository.UserRepository,
) *NotificationService {
	return &NotificationService{repo: repo, invRepo: invRepo, userRepo: userRepo}
}

// Create emits a notification, respecting user preferences for configurable types.
func (s *NotificationService) Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	return s.repo.Create(userID, notifType, title, body, metadata, actions)
}

func (s *NotificationService) List(userID int64, limit, offset int) ([]models.Notification, error) {
	return s.repo.List(userID, limit, offset)
}

func (s *NotificationService) UnreadCount(userID int64) (int, error) {
	return s.repo.UnreadCount(userID)
}

func (s *NotificationService) MarkRead(notifID, userID int64) (bool, error) {
	return s.repo.MarkRead(notifID, userID)
}

func (s *NotificationService) MarkAllRead(userID int64) error {
	return s.repo.MarkAllRead(userID)
}

// ExecuteAction dispatches the action for a notification (invitation accept/reject, coach request approve/reject).
func (s *NotificationService) ExecuteAction(notifID, userID int64, action string) error {
	notif, err := s.repo.GetByID(notifID, userID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if len(notif.Actions) == 0 || string(notif.Actions) == "null" {
		return errors.New("no actions available for this notification")
	}

	var actionList []models.NotificationAction
	if err := json.Unmarshal(notif.Actions, &actionList); err != nil {
		return errors.New("invalid actions data")
	}
	valid := false
	for _, a := range actionList {
		if a.Key == action {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("invalid action")
	}

	switch notif.Type {
	case "invitation_received":
		var meta struct {
			InvitationID int64 `json:"invitation_id"`
		}
		if len(notif.Metadata) > 0 {
			_ = json.Unmarshal(notif.Metadata, &meta)
		}
		if meta.InvitationID == 0 {
			return errors.New("missing invitation reference")
		}
		status, err := s.invRepo.GetStatus(meta.InvitationID)
		if err != nil {
			return err
		}
		if status != "pending" {
			_ = s.repo.ClearActions(notifID)
			return ErrInvitationNotPending
		}
		switch action {
		case "accept":
			if err := s.acceptInvitation(meta.InvitationID, userID); err != nil {
				return err
			}
		case "reject":
			s.rejectInvitation(meta.InvitationID, userID)
		}

	case "coach_request":
		var meta struct {
			RequesterID   int64  `json:"requester_id"`
			RequesterName string `json:"requester_name"`
		}
		if len(notif.Metadata) > 0 {
			_ = json.Unmarshal(notif.Metadata, &meta)
		}
		if meta.RequesterID == 0 {
			return errors.New("missing requester reference")
		}
		switch action {
		case "approve":
			s.approveCoachRequest(meta.RequesterID, meta.RequesterName)
		case "reject":
			s.rejectCoachRequest(meta.RequesterID, meta.RequesterName)
		}
		if err := s.repo.ClearCoachRequestActions(meta.RequesterID); err != nil {
			log.Printf("WARN clear coach_request actions: %v", err)
		}

	default:
		return errors.New("unsupported notification type for actions")
	}

	return s.repo.ClearActions(notifID)
}

func (s *NotificationService) GetPreferences(userID int64) (models.NotificationPreferences, error) {
	return s.repo.GetPreferences(userID)
}

func (s *NotificationService) UpdatePreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error {
	return s.repo.UpsertPreferences(userID, req)
}

// AcceptInvitation is exported so InvitationService can call it when responding to invitations directly.
func (s *NotificationService) AcceptInvitation(invitationID, userID int64) error {
	return s.acceptInvitation(invitationID, userID)
}

// RejectInvitation is exported so InvitationService can call it.
func (s *NotificationService) RejectInvitation(invitationID, userID int64) {
	s.rejectInvitation(invitationID, userID)
}

func (s *NotificationService) acceptInvitation(invitationID, userID int64) error {
	_, _, senderID, err := s.invRepo.AcceptTx(invitationID, userID)
	if err != nil {
		return err
	}
	name, _, _ := s.userRepo.GetNameAndAvatar(userID)
	meta := map[string]interface{}{"invitation_id": invitationID, "user_name": name}
	_ = s.repo.Create(senderID, "invitation_accepted", "notif_invitation_accepted_title", "notif_invitation_accepted_body", meta, nil)
	return nil
}

func (s *NotificationService) rejectInvitation(invitationID, userID int64) {
	senderID, err := s.invRepo.Reject(invitationID)
	if err != nil {
		log.Printf("WARN reject invitation: %v", err)
		return
	}
	name, _, _ := s.userRepo.GetNameAndAvatar(userID)
	meta := map[string]interface{}{"invitation_id": invitationID, "user_name": name}
	_ = s.repo.Create(senderID, "invitation_rejected", "notif_invitation_rejected_title", "notif_invitation_rejected_body", meta, nil)
}

func (s *NotificationService) approveCoachRequest(requesterID int64, requesterName string) {
	if err := s.userRepo.ApproveAsCoach(requesterID); err != nil {
		log.Printf("WARN approve coach request: %v", err)
	}
	meta := map[string]interface{}{"requester_name": requesterName}
	_ = s.repo.Create(requesterID, "coach_request_approved", "notif_coach_request_approved_title", "notif_coach_request_approved_body", meta, nil)
}

func (s *NotificationService) rejectCoachRequest(requesterID int64, requesterName string) {
	meta := map[string]interface{}{"requester_name": requesterName}
	_ = s.repo.Create(requesterID, "coach_request_rejected", "notif_coach_request_rejected_title", "notif_coach_request_rejected_body", meta, nil)
}
```

Note: `ErrNotFound` doesn't exist yet. Add it to `services/errors.go`:
```go
var ErrNotFound = errors.New("not found")
```

#### Step 1.7: Slim down `handlers/notification_handler.go`

- [ ] Replace `handlers/notification_handler.go` entirely:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type NotificationHandler struct {
	svc *services.NotificationService
}

func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	notifications, err := h.svc.List(userID, limit, (page-1)*limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}
	writeJSON(w, http.StatusOK, notifications)
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	count, err := h.svc.UnreadCount(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := trimSuffix(r.URL.Path, "/read")
	notifID, err := extractID(path, "/api/notifications/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}
	found, err := h.svc.MarkRead(notifID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Notification marked as read"})
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.svc.MarkAllRead(userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to mark notifications as read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "All notifications marked as read"})
}

func (h *NotificationHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := trimSuffix(r.URL.Path, "/action")
	notifID, err := extractID(path, "/api/notifications/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid notification ID")
		return
	}
	var req models.NotificationActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.ExecuteAction(notifID, userID, req.Action); err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Notification not found")
		case services.ErrInvitationNotPending:
			writeError(w, http.StatusConflict, err.Error())
		case services.ErrStudentMaxCoaches:
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Action executed"})
}

func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	prefs, err := h.svc.GetPreferences(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch preferences")
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.UpdateNotificationPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.UpdatePreferences(userID, req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Preferences updated"})
}
```

Note: `trimSuffix` is `strings.TrimSuffix` — import `"strings"`. Add `trimSuffix` as a local inline or just use `strings.TrimSuffix` directly. Also add `"strings"` import.

#### Step 1.8: Add `trimSuffix` helper to `handlers/helpers.go`

The new slim handlers need to call `strings.TrimSuffix` without importing strings explicitly. Either import `"strings"` in each file, or add a thin wrapper to `helpers.go`. **Preferred**: just import `"strings"` in each handler file as needed. Remove the `trimSuffix` reference from above and use `strings.TrimSuffix` directly.

#### Step 1.9: Update `main.go`

- [ ] In `main.go`, update the `fx.Provide(...)` block:
  - Replace `handlers.NewNotificationHandler,` (which currently takes `db *sql.DB`) with:
    ```go
    repository.NewNotificationRepository,
    repository.NewInvitationRepository,
    services.NewNotificationService,
    handlers.NewNotificationHandler,
    ```
  - Remove the old inline `handlers.NewNotificationHandler` that was receiving `*sql.DB`. Since FX resolves by type, `NewNotificationService` returns `*services.NotificationService`, which `NewNotificationHandler` now receives. **Important**: `NewInvitationRepository` must appear before `NewNotificationService` in `fx.Provide` (or FX handles ordering automatically — it does, but list in logical dependency order for readability).
  - Also note: `handlers.NewCoachHandler`, `handlers.NewInvitationHandler`, `handlers.NewAchievementHandler`, `handlers.NewAssignmentMessageHandler`, `handlers.NewAdminHandler` still take `*sql.DB` and `*NotificationHandler` at this point — they will be updated in Tasks 2-6. FX will still wire them with the old signatures for now.

- [ ] Update `handlers/user_handler.go` `NewUserHandler` remains as-is for Task 1 (still takes `*NotificationHandler`). It will be cleaned in Task 7.

#### Step 1.10: Verify build

- [ ] Run: `go build ./...`
  Expected: PASS (no compile errors)

#### Step 1.11: Commit

```bash
git add repository/interfaces.go repository/notification_repository.go repository/invitation_repository.go \
    repository/user_repository.go services/notification_service.go services/errors.go \
    handlers/notification_handler.go main.go
git commit -m "refactor: extract notification domain (Task 1 Plan C)"
```

---

## Chunk 2: Invitation Domain

### Task 2: Invitation Domain

**Files:**
- Modify: `repository/interfaces.go` — extend `InvitationRepository` with full CRUD
- Modify: `repository/invitation_repository.go` — add CRUD methods
- Create: `services/invitation_service.go`
- Modify: `handlers/invitation_handler.go` — replace `DB *sql.DB, Notification *NotificationHandler` with `svc *services.InvitationService`
- Modify: `main.go` — add `services.NewInvitationService`

#### Step 2.1: Extend `InvitationRepository` interface in `repository/interfaces.go`

- [ ] Replace the `InvitationRepository` interface with the full version:

```go
// InvitationRepository handles all invitation-related database operations.
type InvitationRepository interface {
	// Methods used by NotificationService (defined in Task 1)
	GetStatus(id int64) (status string, err error)
	AcceptTx(invitationID, userID int64) (coachID, studentID, senderID int64, err error)
	Reject(invitationID int64) (senderID int64, err error)

	// CRUD methods for InvitationService
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
```

#### Step 2.2: Add CRUD methods to `repository/invitation_repository.go`

- [ ] Append to `repository/invitation_repository.go`:

```go
func (r *invitationRepository) FindReceiverByID(receiverID int64) (bool, bool, error) {
	var isCoach, coachPublic bool
	err := r.db.QueryRow(
		"SELECT COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE id = ?",
		receiverID,
	).Scan(&isCoach, &coachPublic)
	return isCoach, coachPublic, err
}

func (r *invitationRepository) FindReceiverByEmail(email string) (int64, bool, bool, error) {
	var receiverID int64
	var isCoach, coachPublic bool
	err := r.db.QueryRow(
		"SELECT id, COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE email = ?",
		email,
	).Scan(&receiverID, &isCoach, &coachPublic)
	return receiverID, isCoach, coachPublic, err
}

func (r *invitationRepository) IsSenderCoach(senderID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", senderID).Scan(&isCoach)
	return isCoach, err
}

func (r *invitationRepository) CountPending(userID, otherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM invitations WHERE status = 'pending' AND (
			(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		)
	`, userID, otherID, otherID, userID).Scan(&count)
	return count, err
}

func (r *invitationRepository) CountActiveRelationship(userID, otherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM coach_students WHERE status = 'active' AND (
			(coach_id = ? AND student_id = ?) OR (coach_id = ? AND student_id = ?)
		)
	`, userID, otherID, otherID, userID).Scan(&count)
	return count, err
}

func (r *invitationRepository) CountStudentActiveCoaches(studentID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active'",
		studentID,
	).Scan(&count)
	return count, err
}

func (r *invitationRepository) Create(senderID, receiverID int64, invType, message string) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO invitations (type, sender_id, receiver_id, message, status) VALUES (?, ?, ?, ?, 'pending')",
		invType, senderID, receiverID, message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *invitationRepository) GetByID(id int64) (models.Invitation, error) {
	var inv models.Invitation
	err := r.db.QueryRow(`
		SELECT i.id, i.type, i.sender_id, i.receiver_id, COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
			COALESCE(s.name, ''), COALESCE(s.custom_avatar, ''), COALESCE(rv.name, ''), COALESCE(rv.custom_avatar, '')
		FROM invitations i
		JOIN users s ON s.id = i.sender_id
		JOIN users rv ON rv.id = i.receiver_id
		WHERE i.id = ?
	`, id).Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar)
	return inv, err
}

func (r *invitationRepository) List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
	query := `
		SELECT i.id, i.type, i.sender_id, i.receiver_id, COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
			COALESCE(s.name, ''), COALESCE(s.custom_avatar, ''), COALESCE(rv.name, ''), COALESCE(rv.custom_avatar, '')
		FROM invitations i
		JOIN users s ON s.id = i.sender_id
		JOIN users rv ON rv.id = i.receiver_id
		WHERE 1=1
	`
	args := []interface{}{}

	switch direction {
	case "sent":
		query += " AND i.sender_id = ?"
		args = append(args, userID)
	case "received":
		query += " AND i.receiver_id = ?"
		args = append(args, userID)
	default:
		query += " AND (i.sender_id = ? OR i.receiver_id = ?)"
		args = append(args, userID, userID)
	}
	if status != "" {
		query += " AND i.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY i.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitations := []models.Invitation{}
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
			&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar); err != nil {
			continue
		}
		invitations = append(invitations, inv)
	}
	return invitations, nil
}

func (r *invitationRepository) Cancel(invID int64) error {
	_, err := r.db.Exec("UPDATE invitations SET status = 'cancelled', updated_at = NOW() WHERE id = ?", invID)
	return err
}

func (r *invitationRepository) IsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return isAdmin, err
}
```

#### Step 2.3: Create `services/invitation_service.go`

- [ ] Create `services/invitation_service.go`:

```go
package services

import (
	"database/sql"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

var (
	ErrCannotInviteSelf          = errors.New("cannot_invite_self")
	ErrReceiverNotCoach          = errors.New("receiver_not_coach")
	ErrInvitationAlreadyPending  = errors.New("invitation_already_pending")
	ErrAlreadyConnected          = errors.New("already_connected")
	ErrOnlyReceiver              = errors.New("only the receiver can respond")
	ErrOnlySender                = errors.New("only the sender can cancel")
)

type InvitationService struct {
	repo     repository.InvitationRepository
	notifSvc *NotificationService
	userRepo repository.UserRepository
}

func NewInvitationService(
	repo repository.InvitationRepository,
	notifSvc *NotificationService,
	userRepo repository.UserRepository,
) *InvitationService {
	return &InvitationService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *InvitationService) Create(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error) {
	// Resolve receiver
	var receiverID int64
	var receiverIsCoach, receiverCoachPublic bool
	var err error

	if req.ReceiverID > 0 {
		receiverID = req.ReceiverID
		receiverIsCoach, receiverCoachPublic, err = s.repo.FindReceiverByID(receiverID)
		if err == sql.ErrNoRows {
			return models.Invitation{}, ErrNotFound
		}
	} else {
		receiverID, receiverIsCoach, receiverCoachPublic, err = s.repo.FindReceiverByEmail(req.ReceiverEmail)
		if err == sql.ErrNoRows {
			return models.Invitation{}, ErrNotFound
		}
	}
	if err != nil {
		return models.Invitation{}, err
	}

	if senderID == receiverID {
		return models.Invitation{}, ErrCannotInviteSelf
	}

	// Type-specific validation
	if req.Type == "coach_invite" {
		isCoach, err := s.repo.IsSenderCoach(senderID)
		if err != nil {
			log.Printf("WARN check is_coach for invitation: %v", err)
		}
		if !isCoach {
			return models.Invitation{}, ErrNotCoach
		}
	} else if req.Type == "student_request" {
		if !receiverIsCoach || !receiverCoachPublic {
			return models.Invitation{}, ErrReceiverNotCoach
		}
	} else {
		return models.Invitation{}, errors.New("invalid invitation type")
	}

	// Guard: no pending invitation already exists
	pending, _ := s.repo.CountPending(senderID, receiverID)
	if pending > 0 {
		return models.Invitation{}, ErrInvitationAlreadyPending
	}
	active, _ := s.repo.CountActiveRelationship(senderID, receiverID)
	if active > 0 {
		return models.Invitation{}, ErrAlreadyConnected
	}

	// Determine student ID for max-coaches check
	studentID := receiverID
	if req.Type == "student_request" {
		studentID = senderID
	}
	count, _ := s.repo.CountStudentActiveCoaches(studentID)
	if count >= models.MaxCoachesPerStudent {
		return models.Invitation{}, ErrStudentMaxCoaches
	}

	invID, err := s.repo.Create(senderID, receiverID, req.Type, req.Message)
	if err != nil {
		return models.Invitation{}, err
	}

	// Notify receiver
	senderName, senderAvatar, _ := s.userRepo.GetNameAndAvatar(senderID)
	meta := map[string]interface{}{
		"invitation_id": invID,
		"sender_id":     senderID,
		"sender_name":   senderName,
		"sender_avatar": senderAvatar,
	}
	actions := []models.NotificationAction{
		{Key: "accept", Label: "invitation_accept", Style: "primary"},
		{Key: "reject", Label: "invitation_reject", Style: "danger"},
	}
	title, body := "notif_coach_invite_title", "notif_coach_invite_body"
	if req.Type == "student_request" {
		title, body = "notif_student_request_title", "notif_student_request_body"
	}
	_ = s.notifSvc.Create(receiverID, "invitation_received", title, body, meta, actions)

	return s.repo.GetByID(invID)
}

func (s *InvitationService) List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
	return s.repo.List(userID, status, direction, limit, offset)
}

func (s *InvitationService) GetByID(invID, requestingUserID int64) (models.Invitation, error) {
	inv, err := s.repo.GetByID(invID)
	if err == sql.ErrNoRows {
		return models.Invitation{}, ErrNotFound
	}
	if err != nil {
		return models.Invitation{}, err
	}
	isAdmin, _ := s.repo.IsAdmin(requestingUserID)
	if inv.SenderID != requestingUserID && inv.ReceiverID != requestingUserID && !isAdmin {
		return models.Invitation{}, ErrForbidden
	}
	return inv, nil
}

func (s *InvitationService) Respond(invID, userID int64, action string) error {
	if action != "accepted" && action != "rejected" {
		return errors.New("action must be 'accepted' or 'rejected'")
	}
	inv, err := s.repo.GetByID(invID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	isAdmin, _ := s.repo.IsAdmin(userID)
	if inv.ReceiverID != userID && !isAdmin {
		return ErrOnlyReceiver
	}
	if inv.Status != "pending" {
		return ErrInvitationNotPending
	}

	if action == "accepted" {
		if err := s.notifSvc.AcceptInvitation(invID, userID); err != nil {
			return err
		}
	} else {
		s.notifSvc.RejectInvitation(invID, userID)
	}

	// Nullify actions on related notification
	_ = s.notifSvc.repo.ClearActionsByInvitation(userID, invID)
	return nil
}

func (s *InvitationService) Cancel(invID, userID int64) error {
	inv, err := s.repo.GetByID(invID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	isAdmin, _ := s.repo.IsAdmin(userID)
	if inv.SenderID != userID && !isAdmin {
		return ErrOnlySender
	}
	if inv.Status != "pending" {
		return ErrInvitationNotPending
	}
	if err := s.repo.Cancel(invID); err != nil {
		return err
	}
	_ = s.notifSvc.repo.ClearCancelledInvitation(inv.ReceiverID, invID)
	return nil
}
```

Note: `s.notifSvc.repo` is unexported. Export it or add helper methods to `NotificationService`:
```go
// Add to NotificationService:
func (s *NotificationService) ClearActionsByInvitation(userID, invID int64) error {
    return s.repo.ClearActionsByInvitation(userID, invID)
}
func (s *NotificationService) ClearCancelledInvitation(receiverID, invID int64) error {
    return s.repo.ClearCancelledInvitation(receiverID, invID)
}
```
Then call these in `InvitationService` instead.

#### Step 2.4: Slim down `handlers/invitation_handler.go`

- [ ] Replace `handlers/invitation_handler.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type InvitationHandler struct {
	svc *services.InvitationService
}

func NewInvitationHandler(svc *services.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

func (h *InvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	inv, err := h.svc.Create(userID, req)
	if err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "user_not_found")
		case services.ErrCannotInviteSelf:
			writeError(w, http.StatusBadRequest, "cannot_invite_self")
		case services.ErrNotCoach:
			writeError(w, http.StatusBadRequest, "not_a_coach")
		case services.ErrReceiverNotCoach:
			writeError(w, http.StatusBadRequest, "receiver_not_coach")
		case services.ErrInvitationAlreadyPending:
			writeError(w, http.StatusBadRequest, "invitation_already_pending")
		case services.ErrAlreadyConnected:
			writeError(w, http.StatusBadRequest, "already_connected")
		case services.ErrStudentMaxCoaches:
			writeError(w, http.StatusBadRequest, "student_max_coaches")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to create invitation")
		}
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (h *InvitationHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	status := r.URL.Query().Get("status")
	direction := r.URL.Query().Get("direction")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}
	invitations, err := h.svc.List(userID, status, direction, limit, (page-1)*limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch invitations")
		return
	}
	writeJSON(w, http.StatusOK, invitations)
}

func (h *InvitationHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	invID, err := extractID(r.URL.Path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}
	inv, err := h.svc.GetByID(invID, userID)
	if err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrForbidden:
			writeError(w, http.StatusForbidden, "Access denied")
		default:
			writeError(w, http.StatusInternalServerError, "Failed to fetch invitation")
		}
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (h *InvitationHandler) RespondInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/respond")
	invID, err := extractID(path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}
	var req models.RespondInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.svc.Respond(invID, userID, req.Action); err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrOnlyReceiver:
			writeError(w, http.StatusForbidden, err.Error())
		case services.ErrInvitationNotPending:
			writeError(w, http.StatusConflict, err.Error())
		case services.ErrStudentMaxCoaches:
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation " + req.Action})
}

func (h *InvitationHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	invID, err := extractID(r.URL.Path, "/api/invitations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid invitation ID")
		return
	}
	if err := h.svc.Cancel(invID, userID); err != nil {
		switch err {
		case services.ErrNotFound:
			writeError(w, http.StatusNotFound, "Invitation not found")
		case services.ErrOnlySender:
			writeError(w, http.StatusForbidden, err.Error())
		case services.ErrInvitationNotPending:
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to cancel invitation")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation cancelled"})
}
```

#### Step 2.5: Update `main.go`

- [ ] In `main.go`, update `handlers.NewInvitationHandler` (remove its old `*sql.DB, *NotificationHandler` dep — FX now wires it via `*services.InvitationService`). Add:
  ```go
  services.NewInvitationService,
  ```
  in `fx.Provide`.

#### Step 2.6: Verify build and commit

- [ ] Run: `go build ./...` — Expected: PASS
- [ ] Commit:
```bash
git add repository/interfaces.go repository/invitation_repository.go services/invitation_service.go \
    services/errors.go handlers/invitation_handler.go services/notification_service.go main.go
git commit -m "refactor: extract invitation domain (Task 2 Plan C)"
```

---

## Chunk 3: Achievement Domain

### Task 3: Achievement Domain

**Files:**
- Modify: `repository/interfaces.go` — add `AchievementRepository`
- Create: `repository/achievement_repository.go`
- Create: `services/achievement_service.go`
- Modify: `handlers/achievement_handler.go` — replace `DB, Notification` with `svc *services.AchievementService`
- Modify: `main.go`

#### Step 3.1: Add `AchievementRepository` to `repository/interfaces.go`

```go
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
```

#### Step 3.2: Create `repository/achievement_repository.go`

- [ ] Create `repository/achievement_repository.go`:

```go
package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type achievementRepository struct {
	db *sql.DB
}

func NewAchievementRepository(db *sql.DB) AchievementRepository {
	return &achievementRepository{db: db}
}

func (r *achievementRepository) List(coachID int64) ([]models.CoachAchievement, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), COALESCE(extra_info, ''),
			image_file_id, is_public, is_verified, COALESCE(rejection_reason, ''),
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? ORDER BY event_date DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	achievements := []models.CoachAchievement{}
	for rows.Next() {
		var a models.CoachAchievement
		var verifiedAt sql.NullString
		if err := rows.Scan(&a.ID, &a.CoachID, &a.EventName, &a.EventDate,
			&a.DistanceKm, &a.ResultTime, &a.Position, &a.ExtraInfo,
			&a.ImageFileID, &a.IsPublic, &a.IsVerified, &a.RejectionReason,
			&a.VerifiedBy, &verifiedAt, &a.CreatedAt); err != nil {
			log.Printf("WARN scan achievement row: %v", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

func (r *achievementRepository) Create(coachID int64, req models.CreateAchievementRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO coach_achievements (coach_id, event_name, event_date, distance_km, result_time, position, extra_info, image_file_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, coachID, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *achievementRepository) GetForEdit(achID, coachID int64) (bool, string, error) {
	var isVerified bool
	var rejectionReason sql.NullString
	err := r.db.QueryRow(
		"SELECT is_verified, rejection_reason FROM coach_achievements WHERE id = ? AND coach_id = ?",
		achID, coachID,
	).Scan(&isVerified, &rejectionReason)
	if err != nil {
		return false, "", err
	}
	reason := ""
	if rejectionReason.Valid {
		reason = rejectionReason.String
	}
	return isVerified, reason, nil
}

func (r *achievementRepository) Update(achID, coachID int64, req models.UpdateAchievementRequest) error {
	_, err := r.db.Exec(`
		UPDATE coach_achievements SET event_name = ?, event_date = ?, distance_km = ?, result_time = ?,
			position = ?, extra_info = ?, image_file_id = ?, rejection_reason = NULL
		WHERE id = ? AND coach_id = ?
	`, req.EventName, req.EventDate, req.DistanceKm, req.ResultTime, req.Position, req.ExtraInfo, req.ImageFileID, achID, coachID)
	return err
}

func (r *achievementRepository) Delete(achID, coachID int64) (bool, error) {
	result, err := r.db.Exec(
		"DELETE FROM coach_achievements WHERE id = ? AND coach_id = ? AND rejection_reason IS NOT NULL",
		achID, coachID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *achievementRepository) SetVisibility(achID, coachID int64, isPublic bool) (bool, error) {
	result, err := r.db.Exec(
		"UPDATE coach_achievements SET is_public = ? WHERE id = ? AND coach_id = ?",
		isPublic, achID, coachID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func (r *achievementRepository) IsCoach(userID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	return isCoach, err
}

func (r *achievementRepository) GetAdminIDs() ([]int64, error) {
	rows, err := r.db.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *achievementRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}
```

#### Step 3.3: Create `services/achievement_service.go`

- [ ] Create `services/achievement_service.go`:

```go
package services

import (
	"database/sql"
	"errors"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

var (
	ErrAchievementVerified    = errors.New("cannot edit a verified achievement")
	ErrAchievementNotRejected = errors.New("only rejected achievements can be edited")
)

type AchievementService struct {
	repo     repository.AchievementRepository
	notifSvc *NotificationService
}

func NewAchievementService(repo repository.AchievementRepository, notifSvc *NotificationService) *AchievementService {
	return &AchievementService{repo: repo, notifSvc: notifSvc}
}

func (s *AchievementService) ListMy(coachID int64) ([]models.CoachAchievement, error) {
	achievements, err := s.repo.List(coachID)
	if err != nil {
		return nil, err
	}
	for i := range achievements {
		achievements[i].EventDate = truncateDate(achievements[i].EventDate)
		s.populateImageURL(&achievements[i])
	}
	return achievements, nil
}

func (s *AchievementService) Create(coachID int64, req models.CreateAchievementRequest) (int64, error) {
	isCoach, err := s.repo.IsCoach(coachID)
	if err != nil || !isCoach {
		return 0, ErrNotCoach
	}
	if req.EventName == "" || req.EventDate == "" {
		return 0, errors.New("event_name and event_date are required")
	}
	id, err := s.repo.Create(coachID, req)
	if err != nil {
		return 0, err
	}
	s.notifyAdmins(coachID, id, req.EventName)
	return id, nil
}

func (s *AchievementService) Update(achID, coachID int64, req models.UpdateAchievementRequest) error {
	isVerified, rejectionReason, err := s.repo.GetForEdit(achID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isVerified {
		return ErrAchievementVerified
	}
	if rejectionReason == "" {
		return ErrAchievementNotRejected
	}
	if err := s.repo.Update(achID, coachID, req); err != nil {
		return err
	}
	s.notifyAdmins(coachID, achID, req.EventName)
	return nil
}

func (s *AchievementService) Delete(achID, coachID int64) error {
	found, err := s.repo.Delete(achID, coachID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *AchievementService) SetVisibility(achID, coachID int64, isPublic bool) error {
	found, err := s.repo.SetVisibility(achID, coachID, isPublic)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *AchievementService) populateImageURL(a *models.CoachAchievement) {
	if a.ImageFileID == nil {
		return
	}
	uuid, err := s.repo.GetFileUUID(*a.ImageFileID)
	if err != nil {
		return
	}
	a.ImageURL = "/api/files/" + uuid + "/download"
}

func (s *AchievementService) notifyAdmins(coachID, achID int64, eventName string) {
	// Get coach name — uses GetAdminIDs which is already in the repo
	// For coach name we'd need UserRepository. To keep deps minimal, we call GetAdminIDs and
	// accept that coach_name in notification meta may be empty if userRepo not injected.
	// TODO: inject UserRepository into AchievementService if coach name in notification is needed.
	adminIDs, err := s.repo.GetAdminIDs()
	if err != nil {
		return
	}
	meta := map[string]interface{}{
		"achievement_id": achID,
		"event_name":     eventName,
	}
	for _, adminID := range adminIDs {
		_ = s.notifSvc.Create(adminID, "achievement_pending",
			"notif_achievement_pending_title", "notif_achievement_pending_body", meta, nil)
	}
}
```

Note: To include `coach_name` in the notification metadata (as in the original code), inject `UserRepository` into `AchievementService` and call `GetNameAndAvatar(coachID)`. Add it to the constructor:
```go
type AchievementService struct {
    repo     repository.AchievementRepository
    notifSvc *NotificationService
    userRepo repository.UserRepository
}
func NewAchievementService(repo repository.AchievementRepository, notifSvc *NotificationService, userRepo repository.UserRepository) *AchievementService {
    return &AchievementService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}
```
Then in `notifyAdmins`: `name, _, _ := s.userRepo.GetNameAndAvatar(coachID)` and add `"coach_name": name` to meta.

#### Step 3.4: Slim down `handlers/achievement_handler.go`

- [ ] Replace `handlers/achievement_handler.go` with a thin handler using `*services.AchievementService`. Map errors to HTTP status codes same as current handler.

#### Step 3.5: Update `main.go` + verify + commit

- [ ] Add `repository.NewAchievementRepository, services.NewAchievementService` to `fx.Provide`
- [ ] `go build ./...` — PASS
- [ ] Commit: `"refactor: extract achievement domain (Task 3 Plan C)"`

---

## Chunk 4: AssignmentMessage Domain

### Task 4: AssignmentMessage Domain

**Files:**
- Modify: `repository/interfaces.go` — add `AssignmentMessageRepository`
- Create: `repository/assignment_message_repository.go`
- Create: `services/assignment_message_service.go`
- Modify: `handlers/assignment_message_handler.go`
- Modify: `main.go`

#### Step 4.1: Add `AssignmentMessageRepository` to `repository/interfaces.go`

```go
// AssignmentMessageRepository handles assignment message and assigned workout detail operations.
type AssignmentMessageRepository interface {
	GetParticipants(awID int64) (coachID, studentID int64, status, title string, err error)
	List(awID int64) ([]models.AssignmentMessage, error)
	Create(awID, senderID int64, body string) (models.AssignmentMessage, error)
	MarkRead(awID, userID int64) error
	GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error)
	// FetchSegments returns segments for an assigned workout.
	FetchSegments(awID int64) []models.WorkoutSegment
}
```

#### Step 4.2: Create `repository/assignment_message_repository.go`

- [ ] Create `repository/assignment_message_repository.go`, implementing all interface methods.

Key SQL for `GetAssignedWorkoutDetail` is the big JOIN from `assignment_message_handler.go:250-270`. Copy it exactly.

`FetchSegments` is the same query as `fetchSegments(db, id)` in `coach_handler.go`. Duplicate it here (same query on `assigned_workout_segments`).

```go
package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fitreg/api/models"
)

type assignmentMessageRepository struct {
	db *sql.DB
}

func NewAssignmentMessageRepository(db *sql.DB) AssignmentMessageRepository {
	return &assignmentMessageRepository{db: db}
}

func (r *assignmentMessageRepository) GetParticipants(awID int64) (int64, int64, string, string, error) {
	var coachID, studentID int64
	var status, title string
	err := r.db.QueryRow(
		"SELECT coach_id, student_id, status, title FROM assigned_workouts WHERE id = ?", awID,
	).Scan(&coachID, &studentID, &status, &title)
	return coachID, studentID, status, title, err
}

func (r *assignmentMessageRepository) List(awID int64) ([]models.AssignmentMessage, error) {
	rows, err := r.db.Query(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.assigned_workout_id = ?
		ORDER BY am.created_at ASC
	`, awID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []models.AssignmentMessage{}
	for rows.Next() {
		var m models.AssignmentMessage
		var avatar sql.NullString
		if err := rows.Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
			&m.Body, &m.IsRead, &m.CreatedAt); err != nil {
			return nil, err
		}
		if avatar.Valid {
			m.SenderAvatar = avatar.String
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *assignmentMessageRepository) Create(awID, senderID int64, body string) (models.AssignmentMessage, error) {
	result, err := r.db.Exec(
		"INSERT INTO assignment_messages (assigned_workout_id, sender_id, body) VALUES (?, ?, ?)",
		awID, senderID, body,
	)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	msgID, _ := result.LastInsertId()
	var m models.AssignmentMessage
	var avatar sql.NullString
	err = r.db.QueryRow(`
		SELECT am.id, am.assigned_workout_id, am.sender_id, u.name, u.avatar_url,
			am.body, am.is_read, am.created_at
		FROM assignment_messages am
		JOIN users u ON u.id = am.sender_id
		WHERE am.id = ?
	`, msgID).Scan(&m.ID, &m.AssignedWorkoutID, &m.SenderID, &m.SenderName, &avatar,
		&m.Body, &m.IsRead, &m.CreatedAt)
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	if avatar.Valid {
		m.SenderAvatar = avatar.String
	}
	return m, nil
}

func (r *assignmentMessageRepository) MarkRead(awID, userID int64) error {
	_, err := r.db.Exec(
		"UPDATE assignment_messages SET is_read = TRUE WHERE assigned_workout_id = ? AND sender_id != ?",
		awID, userID,
	)
	return err
}

func (r *assignmentMessageRepository) GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error) {
	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields sql.NullString
	var studentName, coachName string
	err := r.db.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at,
			us.name AS student_name, uc.name AS coach_name,
			(SELECT COUNT(*) FROM assignment_messages am
				WHERE am.assigned_workout_id = aw.id AND am.sender_id != ? AND am.is_read = FALSE) AS unread_message_count
		FROM assigned_workouts aw
		JOIN users us ON us.id = aw.student_id
		JOIN users uc ON uc.id = aw.coach_id
		WHERE aw.id = ? AND (aw.coach_id = ? OR aw.student_id = ?)
	`, userID, awID, userID, userID).Scan(
		&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
		&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
		&aw.ImageFileID, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt,
		&studentName, &coachName, &aw.UnreadMessageCount,
	)
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.CoachName = coachName
	return aw, nil
}

func (r *assignmentMessageRepository) FetchSegments(awID int64) []models.WorkoutSegment {
	rows, err := r.db.Query(`
		SELECT id, assigned_workout_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM assigned_workout_segments
		WHERE assigned_workout_id = ?
		ORDER BY order_index ASC
	`, awID)
	if err != nil {
		return []models.WorkoutSegment{}
	}
	defer rows.Close()
	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			continue
		}
		segments = append(segments, s)
	}
	return segments
}

// truncateDate is defined in repository package (same as handlers/helpers.go).
func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
```

Note: Add `GetFileUUID` to `AssignmentMessageRepository` if you want to populate `ImageURL` in `GetAssignedWorkoutDetail`. The original handler does this (lines 299-302). Add:
```go
GetFileUUID(fileID int64) (string, error)
```
And implement it (same query as in `achievement_repository.go`).

#### Step 4.3: Create `services/assignment_message_service.go`

```go
package services

import (
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type AssignmentMessageService struct {
	repo     repository.AssignmentMessageRepository
	notifSvc *NotificationService
}

func NewAssignmentMessageService(repo repository.AssignmentMessageRepository, notifSvc *NotificationService) *AssignmentMessageService {
	return &AssignmentMessageService{repo: repo, notifSvc: notifSvc}
}

func (s *AssignmentMessageService) ListMessages(awID, userID int64) ([]models.AssignmentMessage, error) {
	coachID, studentID, _, _, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID != coachID && userID != studentID {
		return nil, ErrForbidden
	}
	return s.repo.List(awID)
}

func (s *AssignmentMessageService) SendMessage(awID, senderID int64, body string) (models.AssignmentMessage, error) {
	coachID, studentID, status, title, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return models.AssignmentMessage{}, ErrNotFound
	}
	if err != nil {
		return models.AssignmentMessage{}, err
	}
	if senderID != coachID && senderID != studentID {
		return models.AssignmentMessage{}, ErrForbidden
	}
	if status != "pending" {
		return models.AssignmentMessage{}, errors.New("cannot send messages on a non-pending assignment")
	}

	body = strings.TrimSpace(body)
	charCount := utf8.RuneCountInString(body)
	if charCount < 1 || charCount > 2000 {
		return models.AssignmentMessage{}, errors.New("message body must be between 1 and 2000 characters")
	}

	msg, err := s.repo.Create(awID, senderID, body)
	if err != nil {
		return models.AssignmentMessage{}, err
	}

	recipientID := studentID
	if senderID == coachID {
		recipientID = studentID
	} else {
		recipientID = coachID
	}
	meta := map[string]interface{}{
		"assigned_workout_id": awID,
		"workout_title":       title,
		"sender_id":           senderID,
		"sender_name":         msg.SenderName,
	}
	_ = s.notifSvc.Create(recipientID, "assignment_message",
		"notif_assignment_message_title", "notif_assignment_message_body", meta, nil)

	return msg, nil
}

func (s *AssignmentMessageService) MarkRead(awID, userID int64) error {
	coachID, studentID, _, _, err := s.repo.GetParticipants(awID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if userID != coachID && userID != studentID {
		return ErrForbidden
	}
	return s.repo.MarkRead(awID, userID)
}

func (s *AssignmentMessageService) GetAssignedWorkoutDetail(awID, userID int64) (models.AssignedWorkout, error) {
	aw, err := s.repo.GetAssignedWorkoutDetail(awID, userID)
	if err == sql.ErrNoRows {
		return models.AssignedWorkout{}, ErrNotFound
	}
	if err != nil {
		return models.AssignedWorkout{}, err
	}
	aw.Segments = s.repo.FetchSegments(awID)
	if aw.ImageFileID != nil {
		if uuid, err := s.repo.GetFileUUID(*aw.ImageFileID); err == nil {
			aw.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return aw, nil
}
```

#### Step 4.4: Slim down `handlers/assignment_message_handler.go`

- [ ] Replace with thin handler using `*services.AssignmentMessageService`.

#### Step 4.5: Update `main.go` + verify + commit

- [ ] `go build ./...` — PASS
- [ ] Commit: `"refactor: extract assignment_message domain (Task 4 Plan C)"`

---

## Chunk 5: Coach Domain (Largest)

### Task 5: Coach Domain

**Files:**
- Modify: `repository/interfaces.go` — add `CoachRepository`
- Create: `repository/coach_repository.go`
- Create: `services/coach_service.go`
- Modify: `handlers/coach_handler.go` — replace `DB *sql.DB, Notification *NotificationHandler` with `svc *services.CoachService`
- Modify: `main.go`

#### Step 5.1: Add `CoachRepository` to `repository/interfaces.go`

```go
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
	DeleteAssignedWorkout(awID, coachID int64) error
	GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error)
	UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) (coachID int64, workoutTitle string, err error)
	GetDailySummary(coachID int64, date string) ([]models.DailySummaryItem, error)
	GetUserName(id int64) (string, error)
	FetchSegments(awID int64) []models.WorkoutSegment
	GetFileUUID(fileID int64) (string, error)
}
```

You will also need to add a `models.CoachStudentInfo` type and `models.DailySummaryItem` type in `models/` if they don't exist. Looking at the current handler, these are defined inline. Extract them to `models/coach.go`.

Add to `models/coach.go`:
```go
type CoachStudentInfo struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    AvatarURL string `json:"avatar_url"`
}

type DailySummaryWorkout struct {
    ID              int64                   `json:"id"`
    Title           string                  `json:"title"`
    Type            string                  `json:"type"`
    DistanceKm      float64                 `json:"distance_km"`
    DurationSeconds int                     `json:"duration_seconds"`
    Description     string                  `json:"description"`
    Notes           string                  `json:"notes"`
    Status          string                  `json:"status"`
    ResultTimeSec   *int                    `json:"result_time_seconds"`
    ResultDistKm    *float64                `json:"result_distance_km"`
    ResultHR        *int                    `json:"result_heart_rate"`
    ResultFeeling   *int                    `json:"result_feeling"`
    DueDate         string                  `json:"due_date"`
    Segments        []WorkoutSegment        `json:"segments"`
}

type DailySummaryItem struct {
    StudentID     int64                `json:"student_id"`
    StudentName   string               `json:"student_name"`
    StudentAvatar *string              `json:"student_avatar"`
    Workout       *DailySummaryWorkout `json:"assigned_workout"`
}
```

#### Step 5.2: Create `repository/coach_repository.go`

This is the largest file. It contains all the SQL from `coach_handler.go`. The key methods to implement:

- **`GetStudents(coachID)`**: SELECT users + coach_students JOIN
- **`GetRelationship(csID)`**: SELECT coach_id, student_id, status FROM coach_students
- **`EndRelationship(csID)`**: UPDATE coach_students SET status='finished'
- **`GetStudentWorkouts(studentID)`**: SELECT workouts
- **`ListAssignedWorkouts(coachID, ...)`**: Complex SELECT with optional filters + pagination + count query
- **`CreateAssignedWorkout(coachID, req)`**: INSERT + INSERT segments + SELECT back — returns `models.AssignedWorkout`
- **`GetAssignedWorkout(awID, coachID)`**: SELECT + fetchSegments + populateImageURL
- **`UpdateAssignedWorkout(awID, coachID, req)`**: UPDATE + DELETE old segments + INSERT new + SELECT back
- **`DeleteAssignedWorkout(awID, coachID)`**: DELETE
- **`GetMyAssignedWorkouts(studentID, ...)`**: SELECT where student_id=?
- **`UpdateAssignedWorkoutStatus(awID, studentID, req)`**: UPDATE status + optionally INSERT into workouts on completion — returns coachID + workoutTitle for notification
- **`GetDailySummary(coachID, date)`**: Complex LEFT JOIN + de-dup logic

`FetchSegments(awID)` and `GetFileUUID(fileID)` are the same implementations as in other repos.

The `populateImageURL` logic (currently a method on `CoachHandler`) moves into the repo's helper or is called inline.

- [ ] Create `repository/coach_repository.go` with all methods above. Copy SQL verbatim from `coach_handler.go`. Move `fetchSegments` function body into `FetchSegments` method.

Key complexity in `UpdateAssignedWorkoutStatus`:
```go
// After updating status, if completed: INSERT into workouts table
// Returns coachID and workoutTitle for notification creation in service
func (r *coachRepository) UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) (int64, string, error) {
    result, err := r.db.Exec(`UPDATE assigned_workouts SET status = ?, ...`, ...)
    // check rows affected
    // if status == "completed": fetch aw details, INSERT into workouts
    // fetch coachID + title for notification
    return coachID, workoutTitle, nil
}
```

#### Step 5.3: Create `services/coach_service.go`

```go
package services

import (
    "database/sql"
    "errors"

    "github.com/fitreg/api/models"
    "github.com/fitreg/api/repository"
)

type CoachService struct {
    repo     repository.CoachRepository
    notifSvc *NotificationService
    userRepo repository.UserRepository
}

func NewCoachService(repo repository.CoachRepository, notifSvc *NotificationService, userRepo repository.UserRepository) *CoachService {
    return &CoachService{repo: repo, notifSvc: notifSvc, userRepo: userRepo}
}

func (s *CoachService) ListStudents(coachID int64) ([]models.CoachStudentInfo, error) {
    isCoach, err := s.repo.IsCoach(coachID)
    if err != nil {
        return nil, err
    }
    if !isCoach {
        return nil, ErrNotCoach
    }
    return s.repo.GetStudents(coachID)
}

func (s *CoachService) EndRelationship(csID, userID int64) error {
    coachID, studentID, status, err := s.repo.GetRelationship(csID)
    if err == sql.ErrNoRows {
        return ErrNotFound
    }
    if err != nil {
        return err
    }
    isAdmin, _ := s.repo.IsAdmin(userID)
    if coachID != userID && studentID != userID && !isAdmin {
        return ErrForbidden
    }
    if status != "active" {
        return errors.New("relationship is not active")
    }
    if err := s.repo.EndRelationship(csID); err != nil {
        return err
    }
    // Notify the other party
    otherID := studentID
    if userID == coachID {
        otherID = studentID
    } else {
        otherID = coachID
    }
    name, _, _ := s.userRepo.GetNameAndAvatar(userID)
    meta := map[string]interface{}{"user_id": userID, "user_name": name}
    _ = s.notifSvc.Create(otherID, "relationship_ended", "notif_relationship_ended_title", "notif_relationship_ended_body", meta, nil)
    return nil
}

func (s *CoachService) GetStudentWorkouts(coachID, studentID int64) ([]models.Workout, error) {
    ok, err := s.repo.IsStudentOf(coachID, studentID)
    if err != nil || !ok {
        return nil, ErrForbidden
    }
    return s.repo.GetStudentWorkouts(studentID)
}

func (s *CoachService) ListAssignedWorkouts(coachID int64, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
    return s.repo.ListAssignedWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
}

func (s *CoachService) CreateAssignedWorkout(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
    isCoach, err := s.repo.IsCoach(coachID)
    if err != nil || !isCoach {
        return models.AssignedWorkout{}, ErrNotCoach
    }
    if req.Title == "" {
        return models.AssignedWorkout{}, errors.New("title is required")
    }
    if len(req.Segments) == 0 {
        return models.AssignedWorkout{}, errors.New("at least one segment is required")
    }
    ok, err := s.repo.IsStudentOf(coachID, req.StudentID)
    if err != nil || !ok {
        return models.AssignedWorkout{}, ErrForbidden
    }
    aw, err := s.repo.CreateAssignedWorkout(coachID, req)
    if err != nil {
        return models.AssignedWorkout{}, err
    }
    // Notify student
    coachName, _, _ := s.userRepo.GetNameAndAvatar(coachID)
    meta := map[string]interface{}{
        "workout_id":    aw.ID,
        "workout_title": req.Title,
        "coach_name":    coachName,
    }
    _ = s.notifSvc.Create(req.StudentID, "workout_assigned", "notif_workout_assigned_title", "notif_workout_assigned_body", meta, nil)
    return aw, nil
}

func (s *CoachService) GetAssignedWorkout(awID, coachID int64) (models.AssignedWorkout, error) {
    aw, err := s.repo.GetAssignedWorkout(awID, coachID)
    if err == sql.ErrNoRows {
        return models.AssignedWorkout{}, ErrNotFound
    }
    return aw, err
}

func (s *CoachService) UpdateAssignedWorkout(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
    if len(req.Segments) == 0 {
        return models.AssignedWorkout{}, errors.New("at least one segment is required")
    }
    aw, err := s.repo.UpdateAssignedWorkout(awID, coachID, req)
    if err == sql.ErrNoRows {
        return models.AssignedWorkout{}, ErrNotFound
    }
    return aw, err
}

func (s *CoachService) DeleteAssignedWorkout(awID, coachID int64) error {
    err := s.repo.DeleteAssignedWorkout(awID, coachID)
    if err == sql.ErrNoRows {
        return ErrNotFound
    }
    return err
}

func (s *CoachService) GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
    return s.repo.GetMyAssignedWorkouts(studentID, startDate, endDate)
}

func (s *CoachService) UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) error {
    coachID, workoutTitle, err := s.repo.UpdateAssignedWorkoutStatus(awID, studentID, req)
    if err == sql.ErrNoRows {
        return ErrNotFound
    }
    if err != nil {
        return err
    }
    // Notify coach on complete/skip
    if req.Status == "completed" || req.Status == "skipped" {
        studentName, _, _ := s.userRepo.GetNameAndAvatar(studentID)
        notifType := "workout_completed"
        title, body := "notif_workout_completed_title", "notif_workout_completed_body"
        if req.Status == "skipped" {
            notifType = "workout_skipped"
            title, body = "notif_workout_skipped_title", "notif_workout_skipped_body"
        }
        meta := map[string]interface{}{
            "workout_id":    awID,
            "workout_title": workoutTitle,
            "student_name":  studentName,
        }
        _ = s.notifSvc.Create(coachID, notifType, title, body, meta, nil)
    }
    return nil
}

func (s *CoachService) GetDailySummary(coachID int64, date string) ([]models.DailySummaryItem, error) {
    isCoach, err := s.repo.IsCoach(coachID)
    if err != nil {
        return nil, err
    }
    if !isCoach {
        return nil, ErrNotCoach
    }
    return s.repo.GetDailySummary(coachID, date)
}
```

#### Step 5.4: Slim down `handlers/coach_handler.go`

- [ ] Replace with thin handler using `*services.CoachService`.
- The `AddStudent` method returns 410 Gone — keep it:
  ```go
  func (h *CoachHandler) AddStudent(w http.ResponseWriter, r *http.Request) {
      writeError(w, http.StatusGone, "Use POST /api/invitations to invite students")
  }
  ```

#### Step 5.5: Update `main.go` + verify + commit

- [ ] `go build ./...` — PASS
- [ ] Commit: `"refactor: extract coach domain (Task 5 Plan C)"`

---

## Chunk 6: Admin Domain + UserHandler Cleanup

### Task 6: Admin Domain

**Files:**
- Modify: `repository/interfaces.go` — add `AdminRepository`
- Create: `repository/admin_repository.go`
- Create: `services/admin_service.go`
- Modify: `handlers/admin_handler.go` — replace `DB, Notification` with `svc *services.AdminService`
- Modify: `main.go`

#### Step 6.1: Add `AdminRepository` to `repository/interfaces.go`

```go
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
```

Add new model types to `models/coach.go` or a new `models/admin.go`:
```go
type AdminUser struct {
    ID        int64  `json:"id"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    AvatarURL string `json:"avatar_url"`
    IsCoach   bool   `json:"is_coach"`
    IsAdmin   bool   `json:"is_admin"`
    CreatedAt string `json:"created_at"`
}

type AdminPendingAchievement struct {
    ID          int64   `json:"id"`
    CoachID     int64   `json:"coach_id"`
    EventName   string  `json:"event_name"`
    EventDate   string  `json:"event_date"`
    DistanceKm  float64 `json:"distance_km"`
    ResultTime  string  `json:"result_time"`
    Position    int     `json:"position"`
    ExtraInfo   string  `json:"extra_info"`
    ImageFileID *int64  `json:"image_file_id"`
    ImageURL    string  `json:"image_url,omitempty"`
    CreatedAt   string  `json:"created_at"`
    CoachName   string  `json:"coach_name"`
}
```

#### Step 6.2: Create `repository/admin_repository.go`

- [ ] Implement all interface methods from the SQL in `admin_handler.go`. Key points:
  - `ListUsers` builds the dynamic WHERE clause (same logic as handler: search, role filter, sort whitelist, pagination + count query)
  - `VerifyAchievement` updates `is_verified = TRUE`, then fetches `coach_id, event_name` for notification — return both
  - `RejectAchievement` fetches achievement first (check pending), then updates rejection_reason — return `coachID, eventName`

#### Step 6.3: Create `services/admin_service.go`

```go
package services

import (
    "github.com/fitreg/api/models"
    "github.com/fitreg/api/repository"
)

type AdminService struct {
    repo     repository.AdminRepository
    notifSvc *NotificationService
}

func NewAdminService(repo repository.AdminRepository, notifSvc *NotificationService) *AdminService {
    return &AdminService{repo: repo, notifSvc: notifSvc}
}

func (s *AdminService) GetStats(userID int64) (map[string]int, error) {
    isAdmin, _ := s.repo.IsAdmin(userID)
    if !isAdmin {
        return nil, ErrForbidden
    }
    totalUsers, totalCoaches, totalRatings, pending, err := s.repo.GetStats()
    if err != nil {
        return nil, err
    }
    return map[string]int{
        "total_users":          totalUsers,
        "total_coaches":        totalCoaches,
        "total_ratings":        totalRatings,
        "pending_achievements": pending,
    }, nil
}

func (s *AdminService) ListUsers(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error) {
    isAdmin, _ := s.repo.IsAdmin(userID)
    if !isAdmin {
        return nil, 0, ErrForbidden
    }
    return s.repo.ListUsers(search, role, sortCol, sortOrder, limit, offset)
}

func (s *AdminService) UpdateUser(callerID, targetID int64, isCoach, isAdmin *bool) error {
    ok, _ := s.repo.IsAdmin(callerID)
    if !ok {
        return ErrForbidden
    }
    return s.repo.UpdateUserRoles(targetID, isCoach, isAdmin)
}

func (s *AdminService) ListPendingAchievements(userID int64) ([]models.AdminPendingAchievement, error) {
    isAdmin, _ := s.repo.IsAdmin(userID)
    if !isAdmin {
        return nil, ErrForbidden
    }
    achievements, err := s.repo.ListPendingAchievements()
    if err != nil {
        return nil, err
    }
    for i := range achievements {
        achievements[i].EventDate = truncateDate(achievements[i].EventDate)
        if achievements[i].ImageFileID != nil {
            if uuid, err := s.repo.GetFileUUID(*achievements[i].ImageFileID); err == nil {
                achievements[i].ImageURL = "/api/files/" + uuid + "/download"
            }
        }
    }
    return achievements, nil
}

func (s *AdminService) VerifyAchievement(achID, adminID int64) error {
    isAdmin, _ := s.repo.IsAdmin(adminID)
    if !isAdmin {
        return ErrForbidden
    }
    coachID, eventName, err := s.repo.VerifyAchievement(achID, adminID)
    if err != nil {
        return err
    }
    meta := map[string]interface{}{"achievement_id": achID, "event_name": eventName}
    _ = s.notifSvc.Create(coachID, "achievement_verified", "notif_achievement_verified_title", "notif_achievement_verified_body", meta, nil)
    return nil
}

func (s *AdminService) RejectAchievement(achID, adminID int64, reason string) error {
    isAdmin, _ := s.repo.IsAdmin(adminID)
    if !isAdmin {
        return ErrForbidden
    }
    coachID, eventName, err := s.repo.RejectAchievement(achID, reason)
    if err != nil {
        return err
    }
    meta := map[string]interface{}{
        "achievement_id": achID,
        "event_name":     eventName,
        "reason":         reason,
    }
    _ = s.notifSvc.Create(coachID, "achievement_rejected", "notif_achievement_rejected_title", "notif_achievement_rejected_body", meta, nil)
    return nil
}
```

#### Step 6.4: Slim down `handlers/admin_handler.go`

- [ ] Replace with thin handler using `*services.AdminService`.

#### Step 6.5: Update `main.go` + verify + commit

- [ ] `go build ./...` — PASS
- [ ] Commit: `"refactor: extract admin domain (Task 6 Plan C)"`

---

### Task 7: UserHandler Cleanup

Remove the interim `*NotificationHandler` dep from `UserHandler` and replace with `*NotificationService`.

**Files:**
- Modify: `handlers/user_handler.go` — update `NewUserHandler` and `RequestCoach` to use `*services.NotificationService`
- Modify: `main.go` — remove `handlers.NewNotificationHandler` from `fx.Provide` for `NewUserHandler`'s old dep (FX wires automatically)

#### Step 7.1: Update `handlers/user_handler.go`

- [ ] In `handlers/user_handler.go`, find the struct definition:
  ```go
  type UserHandler struct {
      svc *services.UserService
      nh  *NotificationHandler  // interim
  }
  func NewUserHandler(svc *services.UserService, nh *NotificationHandler) *UserHandler {
  ```
  Replace with:
  ```go
  type UserHandler struct {
      svc      *services.UserService
      notifSvc *services.NotificationService
  }
  func NewUserHandler(svc *services.UserService, notifSvc *services.NotificationService) *UserHandler {
  ```

- [ ] In `RequestCoach` (or wherever `h.nh.CreateNotification(...)` is called), replace with `h.notifSvc.Create(...)`.

#### Step 7.2: Verify + commit

- [ ] `go build ./...` — PASS
- [ ] Commit: `"refactor: clean up UserHandler notification dep (Task 7 Plan C)"`

---

## Final Verification

After all 7 tasks:

- [ ] Run `go build ./...` — PASS
- [ ] Start the server: `export $(cat .env | xargs) && go run main.go`
- [ ] Smoke test key endpoints: notifications, invitations, achievements, assignment messages, coach routes, admin routes
- [ ] No handler struct has `DB *sql.DB` or `Notification *NotificationHandler` fields
- [ ] Commit any final tweaks
