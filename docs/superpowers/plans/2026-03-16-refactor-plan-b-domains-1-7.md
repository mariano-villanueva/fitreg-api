# API Refactor Plan B: Domains 1-7 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate 7 domains (workout, file, auth, user, template, coach_profile, rating) from monolithic handlers to handlers->services->repository with UberFX DI.

**Architecture:** Each domain gets a repository (SQL only, implements interface), a service (business logic), and a slimmed handler (HTTP parsing + service call). FX wires repo->service->handler automatically. The project compiles and all routes work after each task.

**Tech Stack:** Go 1.24, stdlib HTTP, `*sql.DB` MySQL, UberFX (`go.uber.org/fx`), `github.com/golang-jwt/jwt/v5`, `github.com/fitreg/api/providers/storage`

---

## Conventions

- Module path: `github.com/fitreg/api`
- Repository implementations live in `repository/`
- Service implementations live in `services/`
- All repository interfaces live in `repository/interfaces.go` (single file, grown task by task)
- Handler constructors change signature: `New<X>Handler(db *sql.DB)` becomes `New<X>Handler(svc *services.<X>Service)`
- `main.go` gains `fx.Provide` entries for each new repo + service constructor
- After each task: `go build ./...` must pass
- Commit after each task

## main.go Update Pattern

When a domain is migrated, its repo and service are added to `fx.Provide`, and the handler constructor updates to receive the service. Example for workout:

**Before:**
```go
handlers.NewWorkoutHandler,   // takes *sql.DB
```

**After:**
```go
repository.NewWorkoutRepository,  // takes *sql.DB, returns WorkoutRepository
services.NewWorkoutService,       // takes WorkoutRepository, returns *WorkoutService
handlers.NewWorkoutHandler,       // takes *WorkoutService, returns *WorkoutHandler
```

FX resolves this automatically based on return types.

---

## Chunk 1: Workout Domain

### Task 1: Migrate Workout Domain

#### Step 1.1: Create directories

- [ ] Run:
```bash
mkdir -p repository services
```

#### Step 1.2: Create `repository/interfaces.go`

- [ ] Create `repository/interfaces.go` with the WorkoutRepository interface:

```go
package repository

import "github.com/fitreg/api/models"

// WorkoutRepository handles all workout-related database operations.
type WorkoutRepository interface {
	List(userID int64) ([]models.Workout, error)
	GetByID(id int64) (models.Workout, error)
	ExistsByOwner(id, userID int64) bool
	Create(userID int64, req models.CreateWorkoutRequest) (int64, error)
	Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error) // bool = found
	Delete(id, userID int64) (bool, error)
	GetSegments(workoutID int64) ([]models.WorkoutSegment, error)
	ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error
}
```

#### Step 1.3: Create `repository/workout_repository.go`

- [ ] Create `repository/workout_repository.go`:

```go
package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type workoutRepository struct {
	db *sql.DB
}

// NewWorkoutRepository constructs a WorkoutRepository backed by MySQL.
func NewWorkoutRepository(db *sql.DB) WorkoutRepository {
	return &workoutRepository{db: db}
}

func (r *workoutRepository) List(userID int64) ([]models.Workout, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var avgPace, workoutType, notes sql.NullString
		if err := rows.Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
			&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
			return nil, err
		}
		if avgPace.Valid {
			wo.AvgPace = avgPace.String
		}
		if workoutType.Valid {
			wo.Type = workoutType.String
		}
		if notes.Valid {
			wo.Notes = notes.String
		}
		segs, err := r.GetSegments(wo.ID)
		if err != nil {
			log.Printf("ERROR fetch segments for workout %d: %v", wo.ID, err)
			segs = []models.WorkoutSegment{}
		}
		wo.Segments = segs
		workouts = append(workouts, wo)
	}
	return workouts, nil
}

func (r *workoutRepository) GetByID(id int64) (models.Workout, error) {
	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	if err := r.db.QueryRow(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts WHERE id = ?
	`, id).Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
		return wo, err
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
	}
	segs, err := r.GetSegments(id)
	if err != nil {
		log.Printf("ERROR fetch segments for workout %d: %v", id, err)
		segs = []models.WorkoutSegment{}
	}
	wo.Segments = segs
	return wo, nil
}

func (r *workoutRepository) ExistsByOwner(id, userID int64) bool {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM workouts WHERE id = ? AND user_id = ?", id, userID).Scan(&exists)
	return err == nil
}

func (r *workoutRepository) Create(userID int64, req models.CreateWorkoutRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := r.ReplaceSegments(id, req.Segments); err != nil {
		return id, err
	}
	return id, nil
}

func (r *workoutRepository) Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE workouts SET date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?, avg_heart_rate = ?, feeling = ?, type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ?
	`, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes, id, userID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, nil
	}
	if err := r.ReplaceSegments(id, req.Segments); err != nil {
		return true, err
	}
	return true, nil
}

func (r *workoutRepository) Delete(id, userID int64) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *workoutRepository) GetSegments(workoutID int64) ([]models.WorkoutSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
			COALESCE(value, 0), COALESCE(unit, ''), COALESCE(intensity, ''),
			COALESCE(work_value, 0), COALESCE(work_unit, ''), COALESCE(work_intensity, ''),
			COALESCE(rest_value, 0), COALESCE(rest_unit, ''), COALESCE(rest_intensity, '')
		FROM workout_segments WHERE workout_id = ? ORDER BY order_index
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, nil
}

func (r *workoutRepository) ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error {
	if _, err := r.db.Exec("DELETE FROM workout_segments WHERE workout_id = ?", workoutID); err != nil {
		return err
	}
	for i, seg := range segs {
		if _, err := r.db.Exec(`
			INSERT INTO workout_segments (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return nil
}
```


#### Step 1.4: Create `services/workout_service.go`

- [ ] Create `services/workout_service.go`:

```go
package services

import (
	"database/sql"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// WorkoutService contains business logic for the workout domain.
type WorkoutService struct {
	repo repository.WorkoutRepository
}

// NewWorkoutService constructs a WorkoutService.
func NewWorkoutService(repo repository.WorkoutRepository) *WorkoutService {
	return &WorkoutService{repo: repo}
}

// List returns all workouts for a user.
func (s *WorkoutService) List(userID int64) ([]models.Workout, error) {
	return s.repo.List(userID)
}

// GetByID returns a workout if it exists and is owned by the user.
// Returns sql.ErrNoRows if not found or not owned.
func (s *WorkoutService) GetByID(id, userID int64) (models.Workout, error) {
	if !s.repo.ExistsByOwner(id, userID) {
		return models.Workout{}, sql.ErrNoRows
	}
	return s.repo.GetByID(id)
}

// Create validates and creates a workout, returning the full object.
func (s *WorkoutService) Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
	id, err := s.repo.Create(userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	return s.repo.GetByID(id)
}

// Update validates and updates a workout, returning the full object.
// Returns sql.ErrNoRows if not found.
func (s *WorkoutService) Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
	found, err := s.repo.Update(id, userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if !found {
		return models.Workout{}, sql.ErrNoRows
	}
	return s.repo.GetByID(id)
}

// Delete removes a workout. Returns sql.ErrNoRows if not found.
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
```

#### Step 1.5: Slim `handlers/workout_handler.go`

- [ ] Replace the entire content of `handlers/workout_handler.go` with:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type WorkoutHandler struct {
	svc *services.WorkoutService
}

func NewWorkoutHandler(svc *services.WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{svc: svc}
}

func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	workouts, err := h.svc.List(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workouts")
		return
	}

	writeJSON(w, http.StatusOK, workouts)
}

func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}

	wo, err := h.svc.GetByID(id, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workout")
		return
	}

	writeJSON(w, http.StatusOK, wo)
}

func (h *WorkoutHandler) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "date is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	wo, err := h.svc.Create(userID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create workout")
		return
	}

	writeJSON(w, http.StatusCreated, wo)
}

func (h *WorkoutHandler) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}

	var req models.UpdateWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	wo, err := h.svc.Update(id, userID, req)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update workout")
		return
	}

	writeJSON(w, http.StatusOK, wo)
}

func (h *WorkoutHandler) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}

	err = h.svc.Delete(id, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete workout")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}
```

#### Step 1.6: Update `main.go`

- [ ] Add import for `"github.com/fitreg/api/repository"` and `"github.com/fitreg/api/services"` to `main.go`
- [ ] In `fx.Provide(...)`, add before `handlers.NewWorkoutHandler`:

```go
repository.NewWorkoutRepository,
services.NewWorkoutService,
```

The handler line `handlers.NewWorkoutHandler` stays but now FX resolves `*services.WorkoutService` as its dependency instead of `*sql.DB`.

Full updated `fx.Provide` block:
```go
fx.Provide(
    config.Load,
    dbprovider.New,
    storage.New,
    // Workout domain
    repository.NewWorkoutRepository,
    services.NewWorkoutService,
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

#### Step 1.7: Build & verify

- [ ] Run:
```bash
go build ./...
```
Expected: no output (success).

- [ ] Smoke test (server must be running):
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/workouts
```
Expected: `401` (Unauthorized, since no JWT) -- confirms route is wired.

#### Step 1.8: Commit

- [ ] Run:
```bash
git add repository/interfaces.go repository/workout_repository.go services/workout_service.go handlers/workout_handler.go main.go
git commit -m "refactor: migrate workout domain to repository+service"
```

---

## Chunk 2: File and Auth Domains

### Task 2: Migrate File Domain

#### Step 2.1: Update `repository/interfaces.go`

- [ ] Add `FileRepository` interface. Full file after this task:

```go
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
```

#### Step 2.2: Create `repository/file_repository.go`

- [ ] Create `repository/file_repository.go`:

```go
package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fitreg/api/models"
)

type fileRepository struct {
	db *sql.DB
}

// NewFileRepository constructs a FileRepository backed by MySQL.
func NewFileRepository(db *sql.DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(uuid string, userID int64, name, contentType string, size int64, storageKey string) (models.File, error) {
	result, err := r.db.Exec(
		"INSERT INTO files (uuid, user_id, original_name, content_type, size_bytes, storage_key) VALUES (?, ?, ?, ?, ?, ?)",
		uuid, userID, name, contentType, size, storageKey,
	)
	if err != nil {
		return models.File{}, err
	}

	id, _ := result.LastInsertId()
	f := models.File{
		ID:           id,
		UUID:         uuid,
		OriginalName: name,
		ContentType:  contentType,
		SizeBytes:    size,
		URL:          fmt.Sprintf("/api/files/%s/download", uuid),
		CreatedAt:    time.Now(),
	}
	return f, nil
}

func (r *fileRepository) GetByUUID(uuid string) (models.File, error) {
	var f models.File
	err := r.db.QueryRow(
		"SELECT id, uuid, content_type, storage_key, original_name FROM files WHERE uuid = ?",
		uuid,
	).Scan(&f.ID, &f.UUID, &f.ContentType, &f.StorageKey, &f.OriginalName)
	return f, err
}

func (r *fileRepository) GetOwnerAndKey(uuid string) (userID int64, storageKey string, err error) {
	err = r.db.QueryRow(
		"SELECT user_id, storage_key FROM files WHERE uuid = ?",
		uuid,
	).Scan(&userID, &storageKey)
	return
}

func (r *fileRepository) Delete(uuid string) error {
	_, err := r.db.Exec("DELETE FROM files WHERE uuid = ?", uuid)
	return err
}
```

#### Step 2.3: Create `services/file_service.go`

- [ ] Create `services/file_service.go`:

```go
package services

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/providers/storage"
	"github.com/fitreg/api/repository"
)

// FileService contains business logic for the file domain.
type FileService struct {
	repo  repository.FileRepository
	store storage.Storage
}

// NewFileService constructs a FileService.
func NewFileService(repo repository.FileRepository, store storage.Storage) *FileService {
	return &FileService{repo: repo, store: store}
}

// Upload stores a file in cloud storage and creates a DB record.
// On DB failure it attempts a best-effort rollback of the storage upload.
func (s *FileService) Upload(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error) {
	if err := s.store.Upload(ctx, storageKey, file, contentType); err != nil {
		return models.File{}, err
	}

	f, err := s.repo.Create(uuid, userID, originalName, contentType, size, storageKey)
	if err != nil {
		// Best-effort rollback
		if delErr := s.store.Delete(ctx, storageKey); delErr != nil {
			log.Printf("ERROR rolling back storage upload: %v", delErr)
		}
		return models.File{}, err
	}
	return f, nil
}

// Download retrieves a file from storage by UUID.
func (s *FileService) Download(ctx context.Context, uuid string) (contentType string, reader io.ReadCloser, err error) {
	f, err := s.repo.GetByUUID(uuid)
	if err != nil {
		return "", nil, err
	}

	reader, err = s.store.Download(ctx, f.StorageKey)
	if err != nil {
		return "", nil, err
	}
	return f.ContentType, reader, nil
}

// Delete removes a file from storage and DB. Returns an error with message
// "forbidden" if the requesting user is not the owner.
func (s *FileService) Delete(ctx context.Context, uuid string, userID int64) error {
	ownerID, storageKey, err := s.repo.GetOwnerAndKey(uuid)
	if err != nil {
		return err
	}

	if ownerID != userID {
		return errors.New("forbidden")
	}

	if err := s.store.Delete(ctx, storageKey); err != nil {
		return err
	}
	return s.repo.Delete(uuid)
}
```

#### Step 2.4: Slim `handlers/file_handler.go`

- [ ] Replace the entire content of `handlers/file_handler.go` with:

```go
package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/services"
)

const maxFileSize = 5 << 20 // 5MB

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type FileHandler struct {
	svc *services.FileService
}

func NewFileHandler(svc *services.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// Upload handles POST /api/files
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		writeError(w, http.StatusBadRequest, "File too large (max 5MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedContentTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid file type. Allowed: JPG, PNG, WebP")
		return
	}

	if header.Size > maxFileSize {
		writeError(w, http.StatusBadRequest, "File too large (max 5MB)")
		return
	}

	uuid := generateUUID()
	storageKey := fmt.Sprintf("files/%s%s", uuid, ext)

	f, err := h.svc.Upload(r.Context(), uuid, storageKey, file, contentType, header.Filename, header.Size, userID)
	if err != nil {
		log.Printf("ERROR uploading file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

// Download handles GET /api/files/{uuid}/download
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/files/")
	uuid := strings.TrimSuffix(path, "/download")
	if uuid == "" || uuid == path {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	contentType, reader, err := h.svc.Download(r.Context(), uuid)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		log.Printf("ERROR downloading file: %v", err)
		writeError(w, http.StatusNotFound, "File not found in storage")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("ERROR streaming file %s: %v", uuid, err)
	}
}

// Delete handles DELETE /api/files/{uuid}
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	uuid := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	err := h.svc.Delete(r.Context(), uuid, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil && err.Error() == "forbidden" {
		writeError(w, http.StatusForbidden, "Not authorized to delete this file")
		return
	}
	if err != nil {
		log.Printf("ERROR deleting file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete file")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("ERROR generating UUID: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := fmt.Sprintf("%x", b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
```

#### Step 2.5: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
repository.NewFileRepository,
services.NewFileService,
```

Remove `storage.New` from the area near `handlers.NewFileHandler` -- it stays at the top since `FileService` now depends on `storage.Storage`. The handler line `handlers.NewFileHandler` stays but now FX resolves `*services.FileService` as its dependency.

Full updated `fx.Provide` block:
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

#### Step 2.6: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/files` -- expect `401`

#### Step 2.7: Commit

- [ ] Run:
```bash
git add repository/interfaces.go repository/file_repository.go services/file_service.go handlers/file_handler.go main.go
git commit -m "refactor: migrate file domain to repository+service"
```

---

### Task 3: Migrate Auth Domain

> **Special steps:** This task creates shared user projection helpers and the UserRepository, which will also be used by Task 4 (user domain). There is no separate AuthRepository -- AuthService uses UserRepository.

#### Step 3.1: Add `models.UserRow` and `models.UserProfile` to `models/user.go`

- [ ] Append to `models/user.go` (keep existing `User`, `UpdateProfileRequest`, `CalculateAge`):

```go
// UserRow is the DB scan struct for the 18-column user query.
type UserRow struct {
	ID                  int64
	GoogleID            string
	Email               string
	Name                string
	AvatarURL           sql.NullString
	CustomAvatar        sql.NullString
	Sex                 sql.NullString
	BirthDate           sql.NullString
	WeightKg            sql.NullFloat64
	HeightCm            sql.NullInt64
	Language            sql.NullString
	IsCoach             sql.NullBool
	IsAdmin             sql.NullBool
	CoachDescription    sql.NullString
	CoachPublic         sql.NullBool
	OnboardingCompleted sql.NullBool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// UserProfile is the API response shape for user profile data.
type UserProfile struct {
	ID                  int64     `json:"id"`
	GoogleID            string    `json:"google_id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	CustomAvatar        string    `json:"custom_avatar"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	Age                 int       `json:"age"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	HasCoach            bool      `json:"has_coach"`
	CoachID             int64     `json:"coach_id,omitempty"`
	CoachName           string    `json:"coach_name,omitempty"`
	CoachAvatar         string    `json:"coach_avatar,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
```

Note: Add `"database/sql"` to imports in `models/user.go` since `UserRow` uses `sql.NullString` etc.

Full updated `models/user.go`:

```go
package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID                  int64     `json:"id"`
	GoogleID            string    `json:"google_id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name                string  `json:"name"`
	Sex                 string  `json:"sex"`
	BirthDate           string  `json:"birth_date"`
	WeightKg            float64 `json:"weight_kg"`
	HeightCm            int     `json:"height_cm"`
	Language            string  `json:"language"`
	OnboardingCompleted bool    `json:"onboarding_completed"`
}

// CalculateAge returns the age in years given a birth date string (YYYY-MM-DD).
// Returns 0 if the date is empty or invalid.
func CalculateAge(birthDate string) int {
	if birthDate == "" {
		return 0
	}
	bd, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return 0
	}
	now := time.Now()
	age := now.Year() - bd.Year()
	if now.YearDay() < bd.YearDay() {
		age--
	}
	return age
}

// UserRow is the DB scan struct for the 18-column user query.
type UserRow struct {
	ID                  int64
	GoogleID            string
	Email               string
	Name                string
	AvatarURL           sql.NullString
	CustomAvatar        sql.NullString
	Sex                 sql.NullString
	BirthDate           sql.NullString
	WeightKg            sql.NullFloat64
	HeightCm            sql.NullInt64
	Language            sql.NullString
	IsCoach             sql.NullBool
	IsAdmin             sql.NullBool
	CoachDescription    sql.NullString
	CoachPublic         sql.NullBool
	OnboardingCompleted sql.NullBool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// UserProfile is the API response shape for user profile data.
type UserProfile struct {
	ID                  int64     `json:"id"`
	GoogleID            string    `json:"google_id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	AvatarURL           string    `json:"avatar_url"`
	CustomAvatar        string    `json:"custom_avatar"`
	Sex                 string    `json:"sex"`
	BirthDate           string    `json:"birth_date"`
	Age                 int       `json:"age"`
	WeightKg            float64   `json:"weight_kg"`
	HeightCm            int       `json:"height_cm"`
	Language            string    `json:"language"`
	IsCoach             bool      `json:"is_coach"`
	IsAdmin             bool      `json:"is_admin"`
	CoachDescription    string    `json:"coach_description"`
	CoachPublic         bool      `json:"coach_public"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	HasCoach            bool      `json:"has_coach"`
	CoachID             int64     `json:"coach_id,omitempty"`
	CoachName           string    `json:"coach_name,omitempty"`
	CoachAvatar         string    `json:"coach_avatar,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
```

#### Step 3.2: Update `repository/interfaces.go`

- [ ] Add `UserRepository` interface. Full file after this task:

```go
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
```

#### Step 3.3: Create `repository/user_repository.go`

- [ ] Create `repository/user_repository.go`:

```go
package repository

import (
	"database/sql"

	"github.com/fitreg/api/models"
)

type userRepository struct {
	db *sql.DB
}

// NewUserRepository constructs a UserRepository backed by MySQL.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByGoogleID(googleID string) (models.UserRow, error) {
	var row models.UserRow
	err := r.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE google_id = ?
	`, googleID).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func (r *userRepository) Create(googleID, email, name, avatarURL string) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO users (google_id, email, name, avatar_url) VALUES (?, ?, ?, ?)
	`, googleID, email, name, avatarURL)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *userRepository) UpdateOnLogin(googleID, email, name, picture string) error {
	_, err := r.db.Exec(`
		UPDATE users SET email = ?, name = ?, avatar_url = ?, updated_at = NOW() WHERE google_id = ?
	`, email, name, picture, googleID)
	return err
}

func (r *userRepository) GetByID(id int64) (models.UserRow, error) {
	var row models.UserRow
	err := r.db.QueryRow(`
		SELECT id, google_id, email, name, avatar_url, custom_avatar, sex, birth_date, weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&row.ID, &row.GoogleID, &row.Email, &row.Name, &row.AvatarURL, &row.CustomAvatar,
		&row.Sex, &row.BirthDate, &row.WeightKg, &row.HeightCm, &row.Language, &row.IsCoach, &row.IsAdmin, &row.CoachDescription, &row.CoachPublic, &row.OnboardingCompleted, &row.CreatedAt, &row.UpdatedAt,
	)
	return row, err
}

func (r *userRepository) HasActiveCoach(id int64) (bool, error) {
	var hasCoach bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM coach_students WHERE student_id = ? AND status = 'active')", id).Scan(&hasCoach)
	return hasCoach, err
}

func (r *userRepository) GetActiveCoach(studentID int64) (coachID int64, name, avatar string, found bool) {
	var coachAvatar sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '')
		FROM coach_students cs
		JOIN users u ON u.id = cs.coach_id
		WHERE cs.student_id = ? AND cs.status = 'active'
		LIMIT 1
	`, studentID).Scan(&coachID, &name, &coachAvatar)
	if err != nil {
		return 0, "", "", false
	}
	if coachAvatar.Valid {
		avatar = coachAvatar.String
	}
	return coachID, name, avatar, true
}

func (r *userRepository) UpdateProfile(id int64, req models.UpdateProfileRequest) error {
	var birthDate interface{} = req.BirthDate
	if req.BirthDate == "" {
		birthDate = nil
	}
	var sex interface{} = req.Sex
	if req.Sex == "" {
		sex = nil
	}

	_, err := r.db.Exec(`
		UPDATE users SET name = ?, sex = ?, birth_date = ?, weight_kg = ?, height_cm = ?, language = ?, onboarding_completed = ?, updated_at = NOW() WHERE id = ?
	`, req.Name, sex, birthDate, req.WeightKg, req.HeightCm, req.Language, req.OnboardingCompleted, id)
	return err
}

func (r *userRepository) IsCoach(id int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", id).Scan(&isCoach)
	return isCoach, err
}

func (r *userRepository) HasPendingCoachRequest(id int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE type = 'coach_request' AND actions IS NOT NULL
		AND JSON_EXTRACT(metadata, '$.requester_id') = ?
	`, id).Scan(&count)
	return count > 0, err
}

func (r *userRepository) SetCoachLocality(id int64, locality, level string) error {
	_, err := r.db.Exec("UPDATE users SET coach_locality = ?, coach_level = ?, updated_at = NOW() WHERE id = ?",
		locality, level, id)
	return err
}

func (r *userRepository) GetNameAndAvatar(id int64) (name, avatar string, err error) {
	err = r.db.QueryRow("SELECT COALESCE(name, ''), COALESCE(custom_avatar, '') FROM users WHERE id = ?", id).Scan(&name, &avatar)
	return
}

func (r *userRepository) GetAdminIDs() ([]int64, error) {
	rows, err := r.db.Query("SELECT id FROM users WHERE is_admin = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *userRepository) UploadAvatar(id int64, image string) error {
	_, err := r.db.Exec("UPDATE users SET custom_avatar = ?, updated_at = NOW() WHERE id = ?", image, id)
	return err
}

func (r *userRepository) DeleteAvatar(id int64) error {
	_, err := r.db.Exec("UPDATE users SET custom_avatar = NULL, updated_at = NOW() WHERE id = ?", id)
	return err
}
```


#### Step 3.4: Create `services/user_projection.go`

- [ ] Create `services/user_projection.go` with shared helpers used by both AuthService and UserService:

```go
package services

import (
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// rowToUserProfile converts a models.UserRow (DB scan) to a models.UserProfile (API response).
func rowToUserProfile(row models.UserRow) models.UserProfile {
	u := models.UserProfile{
		ID:        row.ID,
		GoogleID:  row.GoogleID,
		Email:     row.Email,
		Name:      row.Name,
		Language:  "es",
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.CustomAvatar.Valid {
		u.CustomAvatar = row.CustomAvatar.String
		u.AvatarURL = row.CustomAvatar.String
	}
	if row.Sex.Valid {
		u.Sex = row.Sex.String
	}
	if row.BirthDate.Valid {
		bd := truncateDate(row.BirthDate.String)
		u.BirthDate = bd
		u.Age = models.CalculateAge(bd)
	}
	if row.WeightKg.Valid {
		u.WeightKg = row.WeightKg.Float64
	}
	if row.HeightCm.Valid {
		u.HeightCm = int(row.HeightCm.Int64)
	}
	if row.Language.Valid {
		u.Language = row.Language.String
	}
	if row.IsCoach.Valid {
		u.IsCoach = row.IsCoach.Bool
	}
	if row.IsAdmin.Valid {
		u.IsAdmin = row.IsAdmin.Bool
	}
	if row.CoachDescription.Valid {
		u.CoachDescription = row.CoachDescription.String
	}
	if row.CoachPublic.Valid {
		u.CoachPublic = row.CoachPublic.Bool
	}
	if row.OnboardingCompleted.Valid {
		u.OnboardingCompleted = row.OnboardingCompleted.Bool
	}
	return u
}

// fillCoachInfo populates coach fields on a UserProfile if the user has an active coach.
func fillCoachInfo(repo repository.UserRepository, studentID int64, u *models.UserProfile) {
	coachID, name, avatar, found := repo.GetActiveCoach(studentID)
	if !found {
		return
	}
	u.CoachID = coachID
	u.CoachName = name
	u.CoachAvatar = avatar
}

// truncateDate truncates a datetime string to YYYY-MM-DD.
func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
```

#### Step 3.5: Create `services/auth_service.go`

- [ ] Create `services/auth_service.go`:

```go
package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
	"github.com/golang-jwt/jwt/v5"
)

// GoogleTokenInfo holds data from Google's token verification endpoint.
type GoogleTokenInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Aud     string `json:"aud"`
}

// AuthResponse is the response returned after successful authentication.
type AuthResponse struct {
	Token string             `json:"token"`
	User  *models.UserProfile `json:"user"`
}

// AuthService contains business logic for authentication.
type AuthService struct {
	repo           repository.UserRepository
	googleClientID string
	jwtSecret      string
}

// NewAuthService constructs an AuthService.
func NewAuthService(repo repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:           repo,
		googleClientID: cfg.GoogleClientID,
		jwtSecret:      cfg.JWTSecret,
	}
}

// GoogleLogin verifies a Google credential token, finds or creates the user,
// and returns a JWT + user profile.
func (s *AuthService) GoogleLogin(credential string) (*AuthResponse, error) {
	if credential == "" {
		return nil, fmt.Errorf("credential is required")
	}

	tokenInfo, err := s.verifyGoogleToken(credential)
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	if tokenInfo.Aud != s.googleClientID {
		return nil, fmt.Errorf("token audience mismatch")
	}

	user, err := s.findOrCreateUser(tokenInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to process user: %w", err)
	}

	token, err := s.generateJWT(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// verifyGoogleToken makes an outbound HTTP call to Google's tokeninfo endpoint.
// Tech debt: this outbound HTTP call ideally belongs behind a provider interface for testability.
// Acceptable for this refactor phase; can be extracted in a future improvement.
func (s *AuthService) verifyGoogleToken(idToken string) (*GoogleTokenInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token verification failed: %s", string(body))
	}

	var tokenInfo GoogleTokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	return &tokenInfo, nil
}

func (s *AuthService) findOrCreateUser(tokenInfo *GoogleTokenInfo) (*models.UserProfile, error) {
	row, err := s.repo.FindByGoogleID(tokenInfo.Sub)

	if err == nil {
		// Existing user -- update login info
		if err := s.repo.UpdateOnLogin(tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture); err != nil {
			log.Printf("ERROR update user on login: %v", err)
		}

		row.Email = tokenInfo.Email
		row.Name = tokenInfo.Name
		row.AvatarURL = sql.NullString{String: tokenInfo.Picture, Valid: tokenInfo.Picture != ""}
		u := rowToUserProfile(row)
		hasCoach, err := s.repo.HasActiveCoach(row.ID)
		if err != nil {
			log.Printf("ERROR check has coach on login: %v", err)
		}
		u.HasCoach = hasCoach
		if hasCoach {
			fillCoachInfo(s.repo, row.ID, &u)
		}
		return &u, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	id, err := s.repo.Create(tokenInfo.Sub, tokenInfo.Email, tokenInfo.Name, tokenInfo.Picture)
	if err != nil {
		return nil, err
	}

	row, err = s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	u := rowToUserProfile(row)
	return &u, nil
}

func (s *AuthService) generateJWT(userID int64, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
```

#### Step 3.6: Slim `handlers/auth_handler.go`

- [ ] Replace the entire content of `handlers/auth_handler.go` with:

```go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/fitreg/api/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type GoogleLoginRequest struct {
	Credential string `json:"credential"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required")
		return
	}

	resp, err := h.svc.GoogleLogin(req.Credential)
	if err != nil {
		log.Printf("ERROR GoogleLogin: %v", err)
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
```

**Important:** Remove `userRow`, `userJSON`, `rowToJSON`, `fillCoachInfo`, `findOrCreateUser`, `verifyGoogleToken`, `generateJWT` from `handlers/auth_handler.go`. They are now in `services/auth_service.go`, `services/user_projection.go`, and `models/user.go`.

#### Step 3.7: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
repository.NewUserRepository,
services.NewAuthService,
```

Full updated `fx.Provide` block:
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
    // Auth + User domain (shared UserRepository)
    repository.NewUserRepository,
    services.NewAuthService,
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

#### Step 3.8: Fix `handlers/user_handler.go` compile (temporary)

- [ ] The existing `handlers/user_handler.go` still references `userRow`, `rowToJSON`, and `fillCoachInfo` which were removed from `handlers/auth_handler.go`. **For now**, keep a temporary copy of these in `user_handler.go` so it compiles. Add these at the bottom of `handlers/user_handler.go`:

```go
// TEMPORARY: These will be removed in Task 4 when UserHandler is migrated to use UserService.
// They are duplicated here to keep the build passing after Task 3 removed them from auth_handler.go.

type userRow = models.UserRow // alias to the new models type

func rowToJSON(row models.UserRow) models.UserProfile {
	u := models.UserProfile{
		ID:        row.ID,
		GoogleID:  row.GoogleID,
		Email:     row.Email,
		Name:      row.Name,
		Language:  "es",
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.CustomAvatar.Valid {
		u.CustomAvatar = row.CustomAvatar.String
		u.AvatarURL = row.CustomAvatar.String
	}
	if row.Sex.Valid {
		u.Sex = row.Sex.String
	}
	if row.BirthDate.Valid {
		bd := truncateDate(row.BirthDate.String)
		u.BirthDate = bd
		u.Age = models.CalculateAge(bd)
	}
	if row.WeightKg.Valid {
		u.WeightKg = row.WeightKg.Float64
	}
	if row.HeightCm.Valid {
		u.HeightCm = int(row.HeightCm.Int64)
	}
	if row.Language.Valid {
		u.Language = row.Language.String
	}
	if row.IsCoach.Valid {
		u.IsCoach = row.IsCoach.Bool
	}
	if row.IsAdmin.Valid {
		u.IsAdmin = row.IsAdmin.Bool
	}
	if row.CoachDescription.Valid {
		u.CoachDescription = row.CoachDescription.String
	}
	if row.CoachPublic.Valid {
		u.CoachPublic = row.CoachPublic.Bool
	}
	if row.OnboardingCompleted.Valid {
		u.OnboardingCompleted = row.OnboardingCompleted.Bool
	}
	return u
}

func fillCoachInfo(db *sql.DB, studentID int64, u *models.UserProfile) {
	var coachID int64
	var coachName string
	var coachAvatar sql.NullString
	err := db.QueryRow(`
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '')
		FROM coach_students cs
		JOIN users u ON u.id = cs.coach_id
		WHERE cs.student_id = ? AND cs.status = 'active'
		LIMIT 1
	`, studentID).Scan(&coachID, &coachName, &coachAvatar)
	if err != nil {
		logErr("fetch coach info for user", err)
		return
	}
	u.CoachID = coachID
	u.CoachName = coachName
	if coachAvatar.Valid {
		u.CoachAvatar = coachAvatar.String
	}
}
```

Also update the `GetProfile` and `UpdateProfile` methods in `handlers/user_handler.go` to use `models.UserRow` for the variable type and `models.UserProfile` for `rowToJSON` return. Since `userRow` is aliased to `models.UserRow`, existing code using `var row userRow` will continue to work. The `rowToJSON` function now returns `models.UserProfile` instead of `userJSON`.

Update `writeJSON(w, http.StatusOK, rowToJSON(row))` calls -- these now return `models.UserProfile` which has the same JSON tags as the old `userJSON`, so the API response is unchanged.

#### Step 3.9: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/auth/google` -- expect `400`

#### Step 3.10: Commit

- [ ] Run:
```bash
git add models/user.go repository/interfaces.go repository/user_repository.go services/user_projection.go services/auth_service.go handlers/auth_handler.go handlers/user_handler.go main.go
git commit -m "refactor: migrate auth domain to service, create shared UserRepository"
```

---

## Chunk 3: User and Template Domains

### Task 4: Migrate User Domain

> **Key decision:** `UserHandler` keeps `nh *handlers.NotificationHandler` for `RequestCoach`. `UserService` does NOT call notifications. The handler calls `h.nh.CreateNotification(...)` directly. This is documented as an interim state until Plan C migrates NotificationHandler.

#### Step 4.1: Create `services/user_service.go`

- [ ] Create `services/user_service.go`:

```go
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
```

#### Step 4.2: Slim `handlers/user_handler.go`

- [ ] Replace the entire content of `handlers/user_handler.go` with:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type UserHandler struct {
	svc *services.UserService
	nh  *NotificationHandler // interim: kept until Plan C migrates NotificationHandler
}

func NewUserHandler(svc *services.UserService, nh *NotificationHandler) *UserHandler {
	return &UserHandler{svc: svc, nh: nh}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	u, err := h.svc.GetProfile(userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	u, err := h.svc.UpdateProfile(userID, req)
	if err != nil {
		log.Printf("ERROR UpdateProfile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) RequestCoach(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Locality string   `json:"locality"`
		Level    []string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Level) == 0 {
		writeError(w, http.StatusBadRequest, "At least one level is required")
		return
	}

	isCoach, err := h.svc.IsCoach(userID)
	if err != nil {
		logErr("check is coach for request", err)
	}
	if isCoach {
		writeError(w, http.StatusConflict, "User is already a coach")
		return
	}

	pending, err := h.svc.HasPendingCoachRequest(userID)
	if err != nil {
		logErr("check pending coach request count", err)
	}
	if pending {
		writeError(w, http.StatusConflict, "Coach request already pending")
		return
	}

	levelStr := strings.Join(req.Level, ",")
	if err := h.svc.SetCoachLocality(userID, req.Locality, levelStr); err != nil {
		logErr("update user coach locality and level", err)
	}

	requesterName, requesterAvatar, err := h.svc.GetNameAndAvatar(userID)
	if err != nil {
		logErr("fetch requester name for coach request", err)
	}

	adminIDs, err := h.svc.GetAdminIDs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch admins")
		return
	}

	if len(adminIDs) == 0 {
		log.Println("WARNING: No admin users found for coach request notification")
		writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
		return
	}

	meta := map[string]interface{}{
		"requester_id":     userID,
		"requester_name":   requesterName,
		"requester_avatar": requesterAvatar,
		"locality":         req.Locality,
		"level":            req.Level,
	}
	actions := []models.NotificationAction{
		{Key: "approve", Label: "notif_coach_request_approve", Style: "primary"},
		{Key: "reject", Label: "notif_coach_request_reject", Style: "danger"},
	}

	// Interim: call NotificationHandler directly (until Plan C)
	for _, adminID := range adminIDs {
		h.nh.CreateNotification(adminID, "coach_request",
			"notif_coach_request_title", "notif_coach_request_body",
			meta, actions)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Coach request sent"})
}

func (h *UserHandler) GetCoachRequestStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	status, err := h.svc.GetCoachRequestStatus(userID)
	if err != nil {
		logErr("get coach request status", err)
		writeError(w, http.StatusInternalServerError, "Failed to check status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

const maxAvatarSize = 500 * 1024 // 500KB base64

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Image == "" {
		writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	if !strings.HasPrefix(req.Image, "data:image/") {
		writeError(w, http.StatusBadRequest, "image must be a base64 data URI")
		return
	}

	if len(req.Image) > maxAvatarSize {
		writeError(w, http.StatusBadRequest, "image too large (max 500KB)")
		return
	}

	if err := h.svc.UploadAvatar(userID, req.Image); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar updated"})
}

func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := h.svc.DeleteAvatar(userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete avatar")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Avatar removed"})
}
```

#### Step 4.3: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
services.NewUserService,
```

Full updated `fx.Provide` block:
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
    // Auth + User domain (shared UserRepository)
    repository.NewUserRepository,
    services.NewAuthService,
    services.NewUserService,
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

#### Step 4.4: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/me` -- expect `401`

#### Step 4.5: Commit

- [ ] Run:
```bash
git add services/user_service.go handlers/user_handler.go main.go
git commit -m "refactor: migrate user domain to service (interim NotificationHandler dep)"
```

---

### Task 5: Migrate Template Domain

#### Step 5.1: Update `repository/interfaces.go`

- [ ] Add `TemplateRepository` interface. Full file after this task:

```go
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
```

#### Step 5.2: Create `repository/template_repository.go`

- [ ] Create `repository/template_repository.go`:

```go
package repository

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/fitreg/api/models"
)

type templateRepository struct {
	db *sql.DB
}

// NewTemplateRepository constructs a TemplateRepository backed by MySQL.
func NewTemplateRepository(db *sql.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Create(coachID int64, req models.CreateTemplateRequest) (int64, error) {
	var expectedFieldsJSON []byte
	var err error
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			log.Printf("ERROR marshal expected fields: %v", err)
		}
	}

	result, err := r.db.Exec(`
		INSERT INTO workout_templates (coach_id, title, description, type, notes, expected_fields)
		VALUES (?, ?, ?, ?, ?, ?)
	`, coachID, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := r.ReplaceSegments(id, req.Segments); err != nil {
		return id, err
	}
	return id, nil
}

func (r *templateRepository) GetByID(id int64) (models.WorkoutTemplate, error) {
	var t models.WorkoutTemplate
	var description, typ, notes, expectedFields sql.NullString
	err := r.db.QueryRow(`
		SELECT id, coach_id, title, description, type, notes, expected_fields, created_at, updated_at
		FROM workout_templates WHERE id = ?
	`, id).Scan(&t.ID, &t.CoachID, &t.Title, &description, &typ, &notes, &expectedFields, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if description.Valid {
		t.Description = description.String
	}
	if typ.Valid {
		t.Type = typ.String
	}
	if notes.Valid {
		t.Notes = notes.String
	}
	if expectedFields.Valid {
		t.ExpectedFields = json.RawMessage(expectedFields.String)
	}

	segs, err := r.GetSegments(id)
	if err != nil {
		log.Printf("ERROR fetch segments for template %d: %v", id, err)
		segs = []models.TemplateSegment{}
	}
	t.Segments = segs
	return t, nil
}

func (r *templateRepository) List(coachID int64) ([]models.WorkoutTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, title, description, type, notes, expected_fields, created_at, updated_at
		FROM workout_templates
		WHERE coach_id = ?
		ORDER BY created_at DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := []models.WorkoutTemplate{}
	for rows.Next() {
		var t models.WorkoutTemplate
		var description, typ, notes, expectedFields sql.NullString
		if err := rows.Scan(&t.ID, &t.CoachID, &t.Title, &description, &typ, &notes, &expectedFields, &t.CreatedAt, &t.UpdatedAt); err != nil {
			log.Printf("ERROR scan template row: %v", err)
			continue
		}
		if description.Valid {
			t.Description = description.String
		}
		if typ.Valid {
			t.Type = typ.String
		}
		if notes.Valid {
			t.Notes = notes.String
		}
		if expectedFields.Valid {
			t.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		templates = append(templates, t)
	}

	for i := range templates {
		segs, err := r.GetSegments(templates[i].ID)
		if err != nil {
			log.Printf("ERROR fetch segments for template %d: %v", templates[i].ID, err)
			segs = []models.TemplateSegment{}
		}
		templates[i].Segments = segs
	}

	return templates, nil
}

func (r *templateRepository) Update(id, coachID int64, req models.CreateTemplateRequest) error {
	var expectedFieldsJSON []byte
	var err error
	if len(req.ExpectedFields) > 0 {
		expectedFieldsJSON, err = json.Marshal(req.ExpectedFields)
		if err != nil {
			log.Printf("ERROR marshal expected fields for template update: %v", err)
		}
	}

	_, err = r.db.Exec(`
		UPDATE workout_templates SET title = ?, description = ?, type = ?, notes = ?, expected_fields = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.Notes, expectedFieldsJSON, id, coachID)
	if err != nil {
		return err
	}

	return r.ReplaceSegments(id, req.Segments)
}

func (r *templateRepository) Delete(id, coachID int64) (bool, error) {
	result, err := r.db.Exec("DELETE FROM workout_templates WHERE id = ? AND coach_id = ?", id, coachID)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func (r *templateRepository) GetSegments(templateID int64) ([]models.TemplateSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, template_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM workout_template_segments
		WHERE template_id = ?
		ORDER BY order_index ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.TemplateSegment{}
	for rows.Next() {
		var s models.TemplateSegment
		if err := rows.Scan(&s.ID, &s.TemplateID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, nil
}

func (r *templateRepository) ReplaceSegments(templateID int64, segs []models.SegmentRequest) error {
	if _, err := r.db.Exec("DELETE FROM workout_template_segments WHERE template_id = ?", templateID); err != nil {
		return err
	}
	for i, seg := range segs {
		if _, err := r.db.Exec(`
			INSERT INTO workout_template_segments
				(template_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, templateID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return nil
}

func (r *templateRepository) GetCoachID(id int64) (int64, error) {
	var coachID int64
	err := r.db.QueryRow("SELECT coach_id FROM workout_templates WHERE id = ?", id).Scan(&coachID)
	return coachID, err
}
```


#### Step 5.3: Create `services/template_service.go`

- [ ] Create `services/template_service.go`:

```go
package services

import (
	"database/sql"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

// TemplateService contains business logic for the template domain.
type TemplateService struct {
	repo     repository.TemplateRepository
	userRepo repository.UserRepository
}

// NewTemplateService constructs a TemplateService.
func NewTemplateService(repo repository.TemplateRepository, userRepo repository.UserRepository) *TemplateService {
	return &TemplateService{repo: repo, userRepo: userRepo}
}

// Create creates a new template. Returns error if user is not a coach.
func (s *TemplateService) Create(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return models.WorkoutTemplate{}, ErrNotCoach
	}

	id, err := s.repo.Create(coachID, req)
	if err != nil {
		return models.WorkoutTemplate{}, err
	}
	return s.repo.GetByID(id)
}

// List returns all templates for a coach. Returns error if user is not a coach.
func (s *TemplateService) List(coachID int64) ([]models.WorkoutTemplate, error) {
	isCoach, err := s.userRepo.IsCoach(coachID)
	if err != nil || !isCoach {
		return nil, ErrNotCoach
	}
	return s.repo.List(coachID)
}

// Get returns a template by ID. Checks ownership.
func (s *TemplateService) Get(id, coachID int64) (models.WorkoutTemplate, error) {
	tmpl, err := s.repo.GetByID(id)
	if err != nil {
		return tmpl, err
	}
	if tmpl.CoachID != coachID {
		return models.WorkoutTemplate{}, sql.ErrNoRows
	}
	return tmpl, nil
}

// Update updates a template. Checks ownership.
func (s *TemplateService) Update(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	ownerID, err := s.repo.GetCoachID(id)
	if err != nil {
		return models.WorkoutTemplate{}, err
	}
	if ownerID != coachID {
		return models.WorkoutTemplate{}, sql.ErrNoRows
	}

	if err := s.repo.Update(id, coachID, req); err != nil {
		return models.WorkoutTemplate{}, err
	}
	return s.repo.GetByID(id)
}

// Delete removes a template. Returns sql.ErrNoRows if not found.
func (s *TemplateService) Delete(id, coachID int64) error {
	found, err := s.repo.Delete(id, coachID)
	if err != nil {
		return err
	}
	if !found {
		return sql.ErrNoRows
	}
	return nil
}
```

- [ ] Create `services/errors.go` for shared service errors:

```go
package services

import "errors"

// ErrNotCoach is returned when a non-coach user attempts a coach-only operation.
var ErrNotCoach = errors.New("user is not a coach")
```

#### Step 5.4: Slim `handlers/template_handler.go`

- [ ] Replace the entire content of `handlers/template_handler.go` with:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type TemplateHandler struct {
	svc *services.TemplateService
}

func NewTemplateHandler(svc *services.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

// Create handles POST /api/coach/templates
func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	tmpl, err := h.svc.Create(userID, req)
	if err == services.ErrNotCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create template")
		return
	}

	writeJSON(w, http.StatusCreated, tmpl)
}

// List handles GET /api/coach/templates
func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	templates, err := h.svc.List(userID)
	if err == services.ErrNotCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch templates")
		return
	}

	writeJSON(w, http.StatusOK, templates)
}

// Get handles GET /api/coach/templates/{id}
func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	tmpl, err := h.svc.Get(id, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch template")
		return
	}

	writeJSON(w, http.StatusOK, tmpl)
}

// Update handles PUT /api/coach/templates/{id}
func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	tmpl, err := h.svc.Update(id, userID, req)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update template")
		return
	}

	writeJSON(w, http.StatusOK, tmpl)
}

// Delete handles DELETE /api/coach/templates/{id}
func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := extractID(r.URL.Path, "/api/coach/templates/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid template ID")
		return
	}

	err = h.svc.Delete(id, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete template")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Template deleted"})
}
```

#### Step 5.5: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
repository.NewTemplateRepository,
services.NewTemplateService,
```

Full updated `fx.Provide` block:
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

#### Step 5.6: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/coach/templates` -- expect `401`

#### Step 5.7: Commit

- [ ] Run:
```bash
git add repository/interfaces.go repository/template_repository.go services/template_service.go services/errors.go handlers/template_handler.go main.go
git commit -m "refactor: migrate template domain to repository+service"
```

---

## Chunk 4: CoachProfile and Rating Domains

### Task 6: Migrate CoachProfile Domain

#### Step 6.1: Update `repository/interfaces.go`

- [ ] Add `CoachProfileRepository` interface. Full file after this task:

```go
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
```

#### Step 6.2: Create `repository/coach_profile_repository.go`

- [ ] Create `repository/coach_profile_repository.go`:

```go
package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type coachProfileRepository struct {
	db *sql.DB
}

// NewCoachProfileRepository constructs a CoachProfileRepository backed by MySQL.
func NewCoachProfileRepository(db *sql.DB) CoachProfileRepository {
	return &coachProfileRepository{db: db}
}

func (r *coachProfileRepository) UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error {
	_, err := r.db.Exec("UPDATE users SET coach_description = ?, coach_public = ?, updated_at = NOW() WHERE id = ?",
		req.CoachDescription, req.CoachPublic, coachID)
	return err
}

func (r *coachProfileRepository) IsCoach(userID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	return isCoach, err
}

func (r *coachProfileRepository) ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
	where := "WHERE u.is_coach = TRUE AND u.coach_public = TRUE"
	args := []interface{}{}

	if search != "" {
		where += " AND (u.name LIKE ? OR u.coach_description LIKE ? OR u.coach_locality LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if locality != "" {
		where += " AND u.coach_locality LIKE ?"
		args = append(args, "%"+locality+"%")
	}
	if level != "" {
		where += " AND FIND_IN_SET(?, u.coach_level) > 0"
		args = append(args, level)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(DISTINCT u.id) FROM users u " + where
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("ERROR count coaches: %v", err)
	}

	query := `
		SELECT u.id, u.name, COALESCE(u.custom_avatar, '') as avatar_url,
			COALESCE(u.coach_description, '') as coach_description,
			COALESCE(u.coach_locality, '') as coach_locality,
			COALESCE(u.coach_level, '') as coach_level,
			COALESCE(AVG(cr.rating), 0) as avg_rating,
			COUNT(cr.id) as rating_count,
			(SELECT COUNT(*) FROM coach_achievements ca WHERE ca.coach_id = u.id AND ca.is_verified = TRUE) as verified_achievements
		FROM users u
		LEFT JOIN coach_ratings cr ON cr.coach_id = u.id
		` + where + `
		GROUP BY u.id ` + coachSortOrder(sortBy) + `
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	coaches := []models.CoachListItem{}
	for rows.Next() {
		var c models.CoachListItem
		if err := rows.Scan(&c.ID, &c.Name, &c.AvatarURL, &c.CoachDescription,
			&c.CoachLocality, &c.CoachLevel,
			&c.AvgRating, &c.RatingCount, &c.VerifiedCount); err != nil {
			log.Printf("ERROR scan coach list row: %v", err)
			continue
		}
		coaches = append(coaches, c)
	}

	return coaches, total, nil
}

func (r *coachProfileRepository) GetCoachProfile(coachID int64) (models.CoachPublicProfile, error) {
	var profile models.CoachPublicProfile
	var avatarURL, description sql.NullString
	err := r.db.QueryRow(`
		SELECT u.id, u.name, u.custom_avatar, u.coach_description,
			COALESCE(AVG(cr.rating), 0) as avg_rating,
			COUNT(cr.id) as rating_count
		FROM users u
		LEFT JOIN coach_ratings cr ON cr.coach_id = u.id
		WHERE u.id = ? AND u.is_coach = TRUE
		GROUP BY u.id
	`, coachID).Scan(&profile.ID, &profile.Name, &avatarURL, &description,
		&profile.AvgRating, &profile.RatingCount)
	if err != nil {
		return profile, err
	}
	if avatarURL.Valid && avatarURL.String != "" {
		profile.AvatarURL = avatarURL.String
	}
	if description.Valid {
		profile.CoachDescription = description.String
	}
	return profile, nil
}

func (r *coachProfileRepository) IsStudentOf(coachID, studentID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ? AND status = 'active'", coachID, studentID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *coachProfileRepository) CountStudents(coachID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM coach_students WHERE coach_id = ? AND status = 'active'", coachID).Scan(&count)
	return count, err
}

func (r *coachProfileRepository) CountVerifiedAchievements(coachID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM coach_achievements WHERE coach_id = ? AND is_verified = TRUE", coachID).Scan(&count)
	return count, err
}

func (r *coachProfileRepository) GetAchievements(coachID int64) ([]models.CoachAchievement, error) {
	rows, err := r.db.Query(`
		SELECT id, coach_id, event_name, event_date, COALESCE(distance_km, 0),
			COALESCE(result_time, ''), COALESCE(position, 0), COALESCE(extra_info, ''),
			image_file_id, is_public, is_verified, COALESCE(rejection_reason, ''),
			COALESCE(verified_by, 0), COALESCE(verified_at, ''), created_at
		FROM coach_achievements WHERE coach_id = ? AND is_public = TRUE ORDER BY event_date DESC
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
			log.Printf("ERROR scan coach achievement row: %v", err)
			continue
		}
		if verifiedAt.Valid {
			a.VerifiedAt = verifiedAt.String
		}
		// truncate event date
		if len(a.EventDate) >= 10 {
			a.EventDate = a.EventDate[:10]
		}
		achievements = append(achievements, a)
	}
	return achievements, nil
}

func (r *coachProfileRepository) GetRatings(coachID int64) ([]models.CoachRating, error) {
	rows, err := r.db.Query(`
		SELECT cr.id, cr.coach_id, cr.student_id, cr.rating, COALESCE(cr.comment, ''),
			u.name as student_name, cr.created_at, cr.updated_at
		FROM coach_ratings cr
		JOIN users u ON u.id = cr.student_id
		WHERE cr.coach_id = ? ORDER BY cr.updated_at DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.CoachRating{}
	for rows.Next() {
		var rt models.CoachRating
		if err := rows.Scan(&rt.ID, &rt.CoachID, &rt.StudentID, &rt.Rating,
			&rt.Comment, &rt.StudentName, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			log.Printf("ERROR scan coach rating row: %v", err)
			continue
		}
		ratings = append(ratings, rt)
	}
	return ratings, nil
}

func (r *coachProfileRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}

// coachSortOrder returns the ORDER BY clause for coach listing.
func coachSortOrder(sortBy string) string {
	switch sortBy {
	case "name":
		return "ORDER BY u.name ASC"
	case "newest":
		return "ORDER BY u.created_at DESC"
	case "oldest":
		return "ORDER BY u.created_at ASC"
	default:
		return "ORDER BY avg_rating DESC"
	}
}
```


#### Step 6.3: Create `services/coach_profile_service.go`

- [ ] Create `services/coach_profile_service.go`:

```go
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
```

#### Step 6.4: Slim `handlers/coach_profile_handler.go`

- [ ] Replace the entire content of `handlers/coach_profile_handler.go` with:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type CoachProfileHandler struct {
	svc *services.CoachProfileService
}

func NewCoachProfileHandler(svc *services.CoachProfileService) *CoachProfileHandler {
	return &CoachProfileHandler{svc: svc}
}

// UpdateCoachProfile handles PUT /api/coach/profile
func (h *CoachProfileHandler) UpdateCoachProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateCoachProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.svc.UpdateProfile(userID, req)
	if err == services.ErrNotCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}
	if err != nil {
		log.Printf("ERROR updating coach profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update coach profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Coach profile updated"})
}

// ListCoaches handles GET /api/coaches
func (h *CoachProfileHandler) ListCoaches(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	locality := r.URL.Query().Get("locality")
	level := r.URL.Query().Get("level")
	sortBy := r.URL.Query().Get("sort")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 12
	}
	offset := (page - 1) * limit

	coaches, total, err := h.svc.ListCoaches(search, locality, level, sortBy, limit, offset)
	if err != nil {
		log.Printf("ERROR listing coaches: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coaches")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  coaches,
		"total": total,
	})
}

// GetCoachProfile handles GET /api/coaches/{id}
func (h *CoachProfileHandler) GetCoachProfile(w http.ResponseWriter, r *http.Request) {
	coachID, err := extractID(r.URL.Path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	profile, err := h.svc.GetCoachProfile(coachID, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Coach not found")
		return
	}
	if err != nil {
		log.Printf("ERROR fetching coach profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch coach profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
```

#### Step 6.5: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
repository.NewCoachProfileRepository,
services.NewCoachProfileService,
```

Full updated `fx.Provide` block:
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

#### Step 6.6: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/coaches` -- expect `401` or `200`

#### Step 6.7: Commit

- [ ] Run:
```bash
git add repository/interfaces.go repository/coach_profile_repository.go services/coach_profile_service.go handlers/coach_profile_handler.go main.go
git commit -m "refactor: migrate coach_profile domain to repository+service"
```

---

### Task 7: Migrate Rating Domain

#### Step 7.1: Update `repository/interfaces.go`

- [ ] Add `RatingRepository` interface. Full file after this task (final state):

```go
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
```

#### Step 7.2: Create `repository/rating_repository.go`

- [ ] Create `repository/rating_repository.go`:

```go
package repository

import (
	"database/sql"
	"log"

	"github.com/fitreg/api/models"
)

type ratingRepository struct {
	db *sql.DB
}

// NewRatingRepository constructs a RatingRepository backed by MySQL.
func NewRatingRepository(db *sql.DB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) IsStudentOf(coachID, studentID int64) (bool, error) {
	var exists int
	err := r.db.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ?", coachID, studentID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *ratingRepository) Upsert(coachID, studentID int64, rating int, comment string) error {
	_, err := r.db.Exec(`
		INSERT INTO coach_ratings (coach_id, student_id, rating, comment)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE rating = VALUES(rating), comment = VALUES(comment), updated_at = NOW()
	`, coachID, studentID, rating, comment)
	return err
}

func (r *ratingRepository) List(coachID int64) ([]models.CoachRating, error) {
	rows, err := r.db.Query(`
		SELECT cr.id, cr.coach_id, cr.student_id, cr.rating, COALESCE(cr.comment, ''),
			u.name as student_name, cr.created_at, cr.updated_at
		FROM coach_ratings cr
		JOIN users u ON u.id = cr.student_id
		WHERE cr.coach_id = ? ORDER BY cr.updated_at DESC
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ratings := []models.CoachRating{}
	for rows.Next() {
		var rt models.CoachRating
		if err := rows.Scan(&rt.ID, &rt.CoachID, &rt.StudentID, &rt.Rating,
			&rt.Comment, &rt.StudentName, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			log.Printf("ERROR scan rating row: %v", err)
			continue
		}
		ratings = append(ratings, rt)
	}
	return ratings, nil
}
```

#### Step 7.3: Create `services/rating_service.go`

- [ ] Create `services/rating_service.go`:

```go
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
```

#### Step 7.4: Slim `handlers/rating_handler.go`

- [ ] Replace the entire content of `handlers/rating_handler.go` with:

```go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type RatingHandler struct {
	svc *services.RatingService
}

func NewRatingHandler(svc *services.RatingService) *RatingHandler {
	return &RatingHandler{svc: svc}
}

// UpsertRating handles POST /api/coaches/{id}/ratings
func (h *RatingHandler) UpsertRating(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/ratings")
	coachID, err := extractID(path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	var req models.UpsertRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = h.svc.Upsert(coachID, userID, req)
	if err == services.ErrNotStudent {
		writeError(w, http.StatusForbidden, "You are not a student of this coach")
		return
	}
	if err == services.ErrInvalidRating {
		writeError(w, http.StatusBadRequest, "Rating must be between 1 and 10")
		return
	}
	if err != nil {
		log.Printf("ERROR upserting rating: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save rating")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Rating saved"})
}

// GetRatings handles GET /api/coaches/{id}/ratings
func (h *RatingHandler) GetRatings(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/ratings")
	coachID, err := extractID(path, "/api/coaches/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid coach ID")
		return
	}

	ratings, err := h.svc.List(coachID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch ratings")
		return
	}

	writeJSON(w, http.StatusOK, ratings)
}
```

#### Step 7.5: Update `main.go`

- [ ] Add to `fx.Provide`:
```go
repository.NewRatingRepository,
services.NewRatingService,
```

Final `fx.Provide` block after all 7 tasks:
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

Final `main.go` imports:
```go
import (
    "context"
    "log"
    "net/http"
    "time"

    "go.uber.org/fx"

    "github.com/fitreg/api/config"
    "github.com/fitreg/api/handlers"
    "github.com/fitreg/api/middleware"
    dbprovider "github.com/fitreg/api/providers/db"
    "github.com/fitreg/api/providers/storage"
    "github.com/fitreg/api/repository"
    "github.com/fitreg/api/router"
    "github.com/fitreg/api/services"
)
```

#### Step 7.6: Build & verify

- [ ] Run: `go build ./...`
- [ ] Smoke test: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/coaches/1/ratings` -- expect `401`

#### Step 7.7: Commit

- [ ] Run:
```bash
git add repository/interfaces.go repository/rating_repository.go services/rating_service.go handlers/rating_handler.go main.go
git commit -m "refactor: migrate rating domain to repository+service"
```

---

## Post-Plan B Summary

After completing all 7 tasks, the project has:

### New files created:
- `repository/interfaces.go` -- all 6 interfaces (Workout, File, User, Template, CoachProfile, Rating)
- `repository/workout_repository.go`
- `repository/file_repository.go`
- `repository/user_repository.go`
- `repository/template_repository.go`
- `repository/coach_profile_repository.go`
- `repository/rating_repository.go`
- `services/workout_service.go`
- `services/file_service.go`
- `services/auth_service.go`
- `services/user_service.go`
- `services/user_projection.go`
- `services/template_service.go`
- `services/coach_profile_service.go`
- `services/rating_service.go`
- `services/errors.go`

### Files modified:
- `models/user.go` -- added `UserRow` and `UserProfile` structs
- `handlers/workout_handler.go` -- slimmed to HTTP-only
- `handlers/file_handler.go` -- slimmed to HTTP-only (keeps `generateUUID`)
- `handlers/auth_handler.go` -- slimmed to HTTP-only
- `handlers/user_handler.go` -- slimmed to HTTP-only (keeps `nh *NotificationHandler` interim dep)
- `handlers/template_handler.go` -- slimmed to HTTP-only
- `handlers/coach_profile_handler.go` -- slimmed to HTTP-only
- `handlers/rating_handler.go` -- slimmed to HTTP-only
- `main.go` -- added repository + services imports and fx.Provide entries

### Files NOT migrated (deferred to Plan C):
- `handlers/notification_handler.go`
- `handlers/coach_handler.go`
- `handlers/invitation_handler.go`
- `handlers/achievement_handler.go`
- `handlers/assignment_message_handler.go`
- `handlers/admin_handler.go`

### Interim state documented:
- `UserHandler` keeps `nh *NotificationHandler` for `RequestCoach` -- will be cleaned up in Plan C
- `coachSortOrder` helper moved to `repository/coach_profile_repository.go` as unexported function
- `generateUUID` stays in `handlers/file_handler.go`
