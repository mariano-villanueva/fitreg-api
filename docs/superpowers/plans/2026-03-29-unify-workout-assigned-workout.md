# Unify Workout + AssignedWorkout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the separate `workouts` and `assigned_workouts` tables/models with a single unified `workouts` table where `coach_id = NULL` means self-assigned.

**Architecture:** Drop both old tables + segments tables + assignment_messages, recreate them unified. Backend: new `Workout` model replaces `AssignedWorkout`; `WorkoutRepository` absorbs assigned-workout CRUD; `CoachRepository` keeps only relationship/summary methods. Frontend: unified `Workout` type; `/api/workouts` serves both personal and assigned; `/api/coach/workouts` replaces `/api/coach/assigned-workouts`; `MyAssignedWorkouts` page deleted.

**Tech Stack:** Go stdlib backend (fx DI), MySQL, React 19 + TypeScript + Vite.

---

## File Map

**Backend — modify:**
- `migrations/001_schema.sql` — add unified schema at end
- `models/workout.go` — full rewrite (new `Workout` struct)
- `models/coach.go` — remove `AssignedWorkout*` request types (keep `WorkoutSegment`, `SegmentRequest`, coach structs)
- `models/assignment_message.go` — rename field `AssignedWorkoutID` → `WorkoutID`
- `repository/interfaces.go` — expand `WorkoutRepository`, shrink `CoachRepository`
- `repository/workout_repository.go` — full rewrite (unified CRUD + coach CRUD)
- `repository/coach_repository.go` — remove assigned-workout methods, update `GetStudentWorkouts` + `GetDailySummary` + `GetWeeklyLoad` queries
- `repository/assignment_message_repository.go` — update all SQL column refs
- `repository/weekly_template_repository.go` — update `Assign()` to target `workouts` table
- `services/workout_service.go` — full rewrite (handles both personal + coach-assigned)
- `services/coach_service.go` — remove assigned-workout methods
- `handlers/interfaces.go` — expand `WorkoutServicer`, shrink `CoachServicer`
- `handlers/workout_handler.go` — full rewrite (add coach endpoints + status endpoint)
- `handlers/coach_handler.go` — remove assigned-workout handlers, keep daily/load/students
- `router/router.go` — swap routes
- `apperr/codes.go` — add `WORKOUT_006` through `WORKOUT_011`

**Frontend — modify:**
- `src/types/index.ts` — replace `AssignedWorkout` with unified `Workout`
- `src/api/workouts.ts` — add `listWorkoutsFiltered`, `updateWorkoutStatus`, `listCoachWorkouts`, `createCoachWorkout`, `updateCoachWorkout`, `deleteCoachWorkout`, `getCoachWorkout`
- `src/api/coach.ts` — remove assigned-workout functions; keep students/daily/load
- `src/pages/WorkoutForm.tsx` — remove mandatory segment validation
- `src/pages/WorkoutList.tsx` — show pending coach-assigned at top, personal in table
- `src/pages/AthleteHome.tsx` — switch from `getMyAssignedWorkouts` → `listWorkouts`
- `src/pages/AssignWorkoutForm.tsx` — update endpoint to `/api/coach/workouts`
- `src/pages/StudentWorkouts.tsx` — endpoint already `/coach/students/{id}/workouts`, no change needed
- `src/components/WeeklyStrip.tsx` — swap `AssignedWorkout` → `Workout`, check `coach_id`
- `src/components/DayModal.tsx` — swap `AssignedWorkout` → `Workout`, update API calls
- `src/components/MonthCalendar.tsx` — swap `getMyAssignedWorkouts` → `listWorkoutsFiltered`
- `src/components/Sidebar.tsx` — remove `/my-assignments` link
- `src/App.tsx` — remove `MyAssignedWorkouts` import + route
- `src/pages/AssignmentDetail.tsx` — update `backTo` links
- `src/pages/Notifications.tsx` — update navigation targets

**Frontend — delete:**
- `src/pages/MyAssignedWorkouts.tsx`

---

## Task 1: DB Migration

**Files:**
- Modify: `migrations/001_schema.sql`

- [ ] **Step 1: Append migration SQL to schema file**

Open `migrations/001_schema.sql` and append at the very end:

```sql
-- ============================================================
-- 2026-03-29: Unify workouts + assigned_workouts
-- ============================================================
DROP TABLE IF EXISTS assigned_workout_segments;
DROP TABLE IF EXISTS workout_segments;
DROP TABLE IF EXISTS assignment_messages;
DROP TABLE IF EXISTS assigned_workouts;
DROP TABLE IF EXISTS workouts;

CREATE TABLE workouts (
  id                  BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id             BIGINT        NOT NULL,
  coach_id            BIGINT        NULL,
  title               VARCHAR(255)  NULL,
  description         TEXT          NULL,
  type                VARCHAR(50)   NULL,
  notes               TEXT          NULL,
  due_date            DATE          NOT NULL,
  distance_km         DECIMAL(10,2) NULL,
  duration_seconds    INT           NULL,
  expected_fields     JSON          NULL,
  result_distance_km  DECIMAL(10,2) NULL,
  result_time_seconds INT           NULL,
  result_heart_rate   INT           NULL,
  result_feeling      INT           NULL,
  avg_pace            VARCHAR(10)   NULL,
  calories            INT           NULL,
  image_file_id       BIGINT        NULL,
  status              ENUM('pending','completed','skipped') NOT NULL DEFAULT 'completed',
  created_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id)       REFERENCES users(id),
  FOREIGN KEY (coach_id)      REFERENCES users(id),
  FOREIGN KEY (image_file_id) REFERENCES files(id)
);

CREATE TABLE workout_segments (
  id             BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  workout_id     BIGINT        NOT NULL,
  order_index    INT           NOT NULL DEFAULT 0,
  segment_type   ENUM('simple','interval') NOT NULL DEFAULT 'simple',
  repetitions    INT           NOT NULL DEFAULT 1,
  value          DECIMAL(10,2) NULL,
  unit           VARCHAR(10)   NULL,
  intensity      VARCHAR(20)   NULL,
  work_value     DECIMAL(10,2) NULL,
  work_unit      VARCHAR(10)   NULL,
  work_intensity VARCHAR(20)   NULL,
  rest_value     DECIMAL(10,2) NULL,
  rest_unit      VARCHAR(10)   NULL,
  rest_intensity VARCHAR(20)   NULL,
  FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE
);

CREATE TABLE assignment_messages (
  id         BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
  workout_id BIGINT   NOT NULL,
  sender_id  BIGINT   NOT NULL,
  body       TEXT     NOT NULL,
  is_read    BOOLEAN  NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
  FOREIGN KEY (sender_id)  REFERENCES users(id)
);
```

- [ ] **Step 2: Run migration against local DB**

```bash
mysql -u root -proot fitreg < ~/Desktop/FitReg/FitRegAPI/migrations/001_schema.sql
```

Expected: no errors. Tables `workouts`, `workout_segments`, `assignment_messages` exist; `assigned_workouts`, `assigned_workout_segments` are gone.

- [ ] **Step 3: Verify schema**

```bash
mysql -u root -proot fitreg -e "DESCRIBE workouts; DESCRIBE workout_segments; DESCRIBE assignment_messages;"
```

Expected: `workouts` has `coach_id`, `due_date`, `status`, `result_*` columns. `workout_segments` has `workout_id`. `assignment_messages` has `workout_id`.

- [ ] **Step 4: Commit**

```bash
cd ~/Desktop/FitReg/FitRegAPI
git add migrations/001_schema.sql
git commit -m "feat: add unified workouts schema migration"
```

---

## Task 2: Backend Models

**Files:**
- Modify: `models/workout.go`
- Modify: `models/coach.go`
- Modify: `models/assignment_message.go`

- [ ] **Step 1: Rewrite `models/workout.go`**

Replace entire file content:

```go
package models

import (
	"encoding/json"
	"time"
)

type Workout struct {
	ID                 int64            `json:"id"`
	UserID             int64            `json:"user_id"`
	CoachID            *int64           `json:"coach_id"`
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	Type               string           `json:"type"`
	Notes              string           `json:"notes"`
	DueDate            string           `json:"due_date"`
	DistanceKm         float64          `json:"distance_km"`
	DurationSeconds    int              `json:"duration_seconds"`
	ExpectedFields     json.RawMessage  `json:"expected_fields"`
	ResultDistanceKm   *float64         `json:"result_distance_km"`
	ResultTimeSeconds  *int             `json:"result_time_seconds"`
	ResultHeartRate    *int             `json:"result_heart_rate"`
	ResultFeeling      *int             `json:"result_feeling"`
	AvgPace            string           `json:"avg_pace"`
	Calories           int              `json:"calories"`
	ImageFileID        *int64           `json:"image_file_id"`
	Status             string           `json:"status"`
	Segments           []WorkoutSegment `json:"segments"`
	ImageURL           string           `json:"image_url,omitempty"`
	UnreadMessageCount int              `json:"unread_message_count"`
	CoachName          string           `json:"coach_name,omitempty"`
	UserName           string           `json:"user_name,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

// CreateWorkoutRequest is used by athletes to log a personal workout.
type CreateWorkoutRequest struct {
	DueDate         string           `json:"due_date"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	AvgPace         string           `json:"avg_pace"`
	Calories        int              `json:"calories"`
	ResultDistanceKm  *float64       `json:"result_distance_km"`
	ResultTimeSeconds *int           `json:"result_time_seconds"`
	ResultHeartRate   *int           `json:"result_heart_rate"`
	ResultFeeling     *int           `json:"result_feeling"`
	Type            string           `json:"type"`
	Notes           string           `json:"notes"`
	Segments        []SegmentRequest `json:"segments"`
}

// UpdateWorkoutRequest is used by athletes to edit a personal workout.
type UpdateWorkoutRequest struct {
	DueDate         string           `json:"due_date"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	AvgPace         string           `json:"avg_pace"`
	Calories        int              `json:"calories"`
	ResultDistanceKm  *float64       `json:"result_distance_km"`
	ResultTimeSeconds *int           `json:"result_time_seconds"`
	ResultHeartRate   *int           `json:"result_heart_rate"`
	ResultFeeling     *int           `json:"result_feeling"`
	Type            string           `json:"type"`
	Notes           string           `json:"notes"`
	Segments        []SegmentRequest `json:"segments"`
}

// UpdateWorkoutStatusRequest is used by athletes to complete/skip a coach-assigned workout.
type UpdateWorkoutStatusRequest struct {
	Status            string   `json:"status"`
	ResultTimeSeconds *int     `json:"result_time_seconds"`
	ResultDistanceKm  *float64 `json:"result_distance_km"`
	ResultHeartRate   *int     `json:"result_heart_rate"`
	ResultFeeling     *int     `json:"result_feeling"`
	ImageFileID       *int64   `json:"image_file_id"`
}

// CreateCoachWorkoutRequest is used by coaches to assign a workout to a student.
type CreateCoachWorkoutRequest struct {
	StudentID       int64            `json:"student_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	ExpectedFields  []string         `json:"expected_fields"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}

// UpdateCoachWorkoutRequest is used by coaches to edit a coach-assigned workout.
type UpdateCoachWorkoutRequest struct {
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Type            string           `json:"type"`
	DistanceKm      float64          `json:"distance_km"`
	DurationSeconds int              `json:"duration_seconds"`
	Notes           string           `json:"notes"`
	ExpectedFields  []string         `json:"expected_fields"`
	DueDate         string           `json:"due_date"`
	Segments        []SegmentRequest `json:"segments"`
}
```

- [ ] **Step 2: Update `models/coach.go` — remove AssignedWorkout types**

Remove lines 19–78 (the `AssignedWorkout`, `CreateAssignedWorkoutRequest`, `UpdateAssignedWorkoutRequest`, `UpdateAssignedWorkoutStatusRequest` structs). Keep everything else: `CoachStudent`, `WorkoutSegment`, `SegmentRequest`, coach profile types, `DailySummaryWorkout`, `DailySummaryItem`, `WeeklyLoadEntry`, `CoachStudentInfo`.

The `WorkoutSegment` struct field `AssignedWorkoutID` (line 82) must be renamed to `WorkoutID` with json tag `"workout_id"`:

```go
type WorkoutSegment struct {
	ID          int64   `json:"id"`
	WorkoutID   int64   `json:"workout_id"`
	OrderIndex  int     `json:"order_index"`
	SegmentType string  `json:"segment_type"`
	Repetitions int     `json:"repetitions"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Intensity   string  `json:"intensity"`
	WorkValue   float64 `json:"work_value"`
	WorkUnit    string  `json:"work_unit"`
	WorkIntensity string `json:"work_intensity"`
	RestValue   float64 `json:"rest_value"`
	RestUnit    string  `json:"rest_unit"`
	RestIntensity string `json:"rest_intensity"`
}
```

- [ ] **Step 3: Update `models/assignment_message.go`**

Rename `AssignedWorkoutID` → `WorkoutID` and update JSON tag:

```go
package models

import "time"

type AssignmentMessage struct {
	ID           int64     `json:"id"`
	WorkoutID    int64     `json:"workout_id"`
	SenderID     int64     `json:"sender_id"`
	SenderName   string    `json:"sender_name"`
	SenderAvatar string    `json:"sender_avatar"`
	Body         string    `json:"body"`
	IsRead       bool      `json:"is_read"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateAssignmentMessageRequest struct {
	Body string `json:"body"`
}
```

- [ ] **Step 4: Build to catch model errors**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go build ./models/...
```

Expected: compilation errors will occur because other packages reference the old fields — that's expected and will be fixed in subsequent tasks.

- [ ] **Step 5: Commit**

```bash
git add models/
git commit -m "feat: unified Workout model replaces AssignedWorkout"
```

---

## Task 3: Backend — apperr codes

**Files:**
- Modify: `apperr/codes.go`

- [ ] **Step 1: Add new workout error codes**

In `apperr/codes.go`, add after `WORKOUT_005`:

```go
	WORKOUT_006 = "WORKOUT_006"
	WORKOUT_007 = "WORKOUT_007"
	WORKOUT_008 = "WORKOUT_008"
	WORKOUT_009 = "WORKOUT_009"
	WORKOUT_010 = "WORKOUT_010"
	WORKOUT_011 = "WORKOUT_011"
```

---

## Task 4: Repository — interfaces.go

**Files:**
- Modify: `repository/interfaces.go`

- [ ] **Step 1: Expand `WorkoutRepository` interface**

Replace the existing `WorkoutRepository` interface with:

```go
// WorkoutRepository handles all workout CRUD for both personal and coach-assigned workouts.
type WorkoutRepository interface {
	// Personal workout methods (coach_id = NULL)
	List(userID int64, startDate, endDate string) ([]models.Workout, error)
	GetByID(id int64) (models.Workout, error)
	Create(userID int64, req models.CreateWorkoutRequest) (int64, error)
	Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error)
	Delete(id, userID int64) (bool, error)

	// Coach-assigned workout methods (coach_id != NULL)
	CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error)
	ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error)
	GetCoachWorkout(workoutID, coachID int64) (models.Workout, error)
	UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error)
	GetWorkoutStatus(workoutID, coachID int64) (string, error)
	DeleteCoachWorkout(workoutID, coachID int64) error

	// Student methods
	GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error)
	UpdateStatus(workoutID, studentID int64, req models.UpdateWorkoutStatusRequest) (coachID int64, workoutTitle string, err error)

	// Shared
	GetSegments(workoutID int64) ([]models.WorkoutSegment, error)
	ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error
	GetFileUUID(fileID int64) (string, error)
}
```

- [ ] **Step 2: Shrink `CoachRepository` — remove assigned-workout methods**

Remove these from the `CoachRepository` interface:
- `ListAssignedWorkouts`
- `CreateAssignedWorkout`
- `GetAssignedWorkout`
- `UpdateAssignedWorkout`
- `GetAssignedWorkoutStatus`
- `DeleteAssignedWorkout`
- `GetMyAssignedWorkouts`
- `UpdateAssignedWorkoutStatus`
- `FetchSegments`
- `GetFileUUID`

Keep:
- `IsCoach`
- `IsAdmin`
- `IsStudentOf`
- `GetStudents`
- `GetRelationship`
- `EndRelationship`
- `GetStudentWorkouts(studentID int64) ([]models.Workout, error)` — returns ALL workouts for student (unified)
- `GetDailySummary`
- `GetUserName`
- `GetWeeklyLoad`

Final `CoachRepository`:

```go
type CoachRepository interface {
	IsCoach(userID int64) (bool, error)
	IsAdmin(userID int64) (bool, error)
	IsStudentOf(coachID, studentID int64) (bool, error)
	GetStudents(coachID int64) ([]models.CoachStudentInfo, error)
	GetRelationship(csID int64) (coachID, studentID int64, status string, err error)
	EndRelationship(csID int64) error
	GetStudentWorkouts(studentID int64) ([]models.Workout, error)
	GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error)
	GetUserName(id int64) (string, error)
	GetWeeklyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}
```

- [ ] **Step 3: Update `AssignmentMessageRepository` interface**

Change `GetAssignedWorkoutDetail` return type from `models.AssignedWorkout` to `models.Workout`:

```go
type AssignmentMessageRepository interface {
	GetParticipants(workoutID int64) (coachID, studentID int64, status, title string, err error)
	List(workoutID int64) ([]models.AssignmentMessage, error)
	Create(workoutID, senderID int64, body string) (models.AssignmentMessage, error)
	MarkRead(workoutID, userID int64) error
	GetWorkoutDetail(workoutID, userID int64) (models.Workout, error)
	GetFileUUID(fileID int64) (string, error)
}
```

---

## Task 5: Repository — workout_repository.go (full rewrite)

**Files:**
- Modify: `repository/workout_repository.go`

- [ ] **Step 1: Replace entire file**

```go
package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fitreg/api/models"
)

type workoutRepository struct {
	db *sql.DB
}

func NewWorkoutRepository(db *sql.DB) WorkoutRepository {
	return &workoutRepository{db: db}
}

// scanWorkout scans a single row into a Workout. The SELECT must return columns in this order:
// id, user_id, coach_id, title, description, type, notes, due_date, distance_km, duration_seconds,
// expected_fields, result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
// avg_pace, calories, image_file_id, status, created_at, updated_at
func scanWorkout(row interface {
	Scan(...interface{}) error
}) (models.Workout, error) {
	var wo models.Workout
	var coachID sql.NullInt64
	var title, description, workoutType, notes, avgPace sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds, calories sql.NullInt64
	var expectedFields sql.NullString
	var resultDistKm sql.NullFloat64
	var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
	var dueDate sql.NullString

	err := row.Scan(
		&wo.ID, &wo.UserID, &coachID,
		&title, &description, &workoutType, &notes, &dueDate,
		&distanceKm, &durationSeconds,
		&expectedFields,
		&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
		&avgPace, &calories, &imageFileID,
		&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
	)
	if err != nil {
		return wo, err
	}
	if coachID.Valid {
		wo.CoachID = &coachID.Int64
	}
	if title.Valid {
		wo.Title = title.String
	}
	if description.Valid {
		wo.Description = description.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
	}
	if dueDate.Valid {
		wo.DueDate = truncateDate(dueDate.String)
	}
	if distanceKm.Valid {
		wo.DistanceKm = distanceKm.Float64
	}
	if durationSeconds.Valid {
		wo.DurationSeconds = int(durationSeconds.Int64)
	}
	if expectedFields.Valid {
		wo.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	if resultDistKm.Valid {
		v := resultDistKm.Float64
		wo.ResultDistanceKm = &v
	}
	if resultTimeSec.Valid {
		v := int(resultTimeSec.Int64)
		wo.ResultTimeSeconds = &v
	}
	if resultHR.Valid {
		v := int(resultHR.Int64)
		wo.ResultHeartRate = &v
	}
	if resultFeeling.Valid {
		v := int(resultFeeling.Int64)
		wo.ResultFeeling = &v
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if calories.Valid {
		wo.Calories = int(calories.Int64)
	}
	if imageFileID.Valid {
		v := imageFileID.Int64
		wo.ImageFileID = &v
	}
	return wo, nil
}

const workoutSelectCols = `
	id, user_id, coach_id, title, description, type, notes, due_date,
	distance_km, duration_seconds, expected_fields,
	result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
	avg_pace, calories, image_file_id, status, created_at, updated_at`

func (r *workoutRepository) List(userID int64, startDate, endDate string) ([]models.Workout, error) {
	query := `SELECT ` + workoutSelectCols + ` FROM workouts WHERE user_id = ?`
	args := []interface{}{userID}
	if startDate != "" {
		query += ` AND due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND due_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY due_date DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkoutRows(rows)
}

func (r *workoutRepository) GetByID(id int64) (models.Workout, error) {
	row := r.db.QueryRow(`SELECT `+workoutSelectCols+` FROM workouts WHERE id = ?`, id)
	return scanWorkout(row)
}

func (r *workoutRepository) Create(userID int64, req models.CreateWorkoutRequest) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO workouts
		  (user_id, coach_id, due_date, distance_km, duration_seconds, avg_pace, calories,
		   result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
		   type, notes, status)
		VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed')
	`, userID, req.DueDate, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories,
		req.ResultDistanceKm, req.ResultTimeSeconds, req.ResultHeartRate, req.ResultFeeling,
		req.Type, req.Notes)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *workoutRepository) Update(id, userID int64, req models.UpdateWorkoutRequest) (bool, error) {
	result, err := r.db.Exec(`
		UPDATE workouts SET
		  due_date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?,
		  result_distance_km = ?, result_time_seconds = ?, result_heart_rate = ?, result_feeling = ?,
		  type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ? AND coach_id IS NULL
	`, req.DueDate, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories,
		req.ResultDistanceKm, req.ResultTimeSeconds, req.ResultHeartRate, req.ResultFeeling,
		req.Type, req.Notes, id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (r *workoutRepository) Delete(id, userID int64) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ? AND coach_id IS NULL`, id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (r *workoutRepository) CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error) {
	var ef interface{}
	if len(req.ExpectedFields) > 0 {
		b, _ := json.Marshal(req.ExpectedFields)
		ef = string(b)
	}
	result, err := r.db.Exec(`
		INSERT INTO workouts
		  (user_id, coach_id, title, description, type, distance_km, duration_seconds, notes,
		   expected_fields, due_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')
	`, req.StudentID, coachID, req.Title, req.Description, req.Type,
		req.DistanceKm, req.DurationSeconds, req.Notes, ef, req.DueDate)
	if err != nil {
		return models.Workout{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.Workout{}, err
	}
	return r.GetByID(id)
}

func (r *workoutRepository) ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error) {
	where := `WHERE w.coach_id = ?`
	args := []interface{}{coachID}
	if studentID != nil {
		where += ` AND w.user_id = ?`
		args = append(args, *studentID)
	}
	if statusFilter != "" {
		where += ` AND w.status = ?`
		args = append(args, statusFilter)
	}
	if startDate != "" {
		where += ` AND w.due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		where += ` AND w.due_date <= ?`
		args = append(args, endDate)
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM workouts w `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT w.` + workoutSelectCols + `, u.name FROM workouts w JOIN users u ON u.id = w.user_id ` + where + ` ORDER BY w.due_date DESC`
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workouts []models.Workout
	for rows.Next() {
		wo, err := scanWorkout(rows)
		if err != nil {
			return nil, 0, err
		}
		if err := rows.Scan(&wo.UserName); err != nil {
			return nil, 0, err
		}
		workouts = append(workouts, wo)
	}
	if workouts == nil {
		workouts = []models.Workout{}
	}
	return workouts, total, nil
}

func (r *workoutRepository) GetCoachWorkout(workoutID, coachID int64) (models.Workout, error) {
	row := r.db.QueryRow(`SELECT `+workoutSelectCols+` FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID)
	return scanWorkout(row)
}

func (r *workoutRepository) UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error) {
	var ef interface{}
	if len(req.ExpectedFields) > 0 {
		b, _ := json.Marshal(req.ExpectedFields)
		ef = string(b)
	}
	_, err := r.db.Exec(`
		UPDATE workouts SET
		  title = ?, description = ?, type = ?, distance_km = ?, duration_seconds = ?,
		  notes = ?, expected_fields = ?, due_date = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds,
		req.Notes, ef, req.DueDate, workoutID, coachID)
	if err != nil {
		return models.Workout{}, err
	}
	return r.GetByID(workoutID)
}

func (r *workoutRepository) GetWorkoutStatus(workoutID, coachID int64) (string, error) {
	var status string
	err := r.db.QueryRow(`SELECT status FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID).Scan(&status)
	return status, err
}

func (r *workoutRepository) DeleteCoachWorkout(workoutID, coachID int64) error {
	result, err := r.db.Exec(`DELETE FROM workouts WHERE id = ? AND coach_id = ?`, workoutID, coachID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *workoutRepository) GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error) {
	query := `SELECT ` + workoutSelectCols + ` FROM workouts WHERE user_id = ?`
	args := []interface{}{studentID}
	if startDate != "" {
		query += ` AND due_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND due_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY due_date ASC`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkoutRows(rows)
}

func (r *workoutRepository) UpdateStatus(workoutID, studentID int64, req models.UpdateWorkoutStatusRequest) (int64, string, error) {
	var coachID int64
	var title string
	var currentStatus string
	err := r.db.QueryRow(
		`SELECT coach_id, COALESCE(title,''), status FROM workouts WHERE id = ? AND user_id = ? AND coach_id IS NOT NULL`,
		workoutID, studentID,
	).Scan(&coachID, &title, &currentStatus)
	if err == sql.ErrNoRows {
		return 0, "", sql.ErrNoRows
	}
	if err != nil {
		return 0, "", err
	}
	if currentStatus != "pending" {
		return 0, "", ErrStatusConflict
	}
	_, err = r.db.Exec(`
		UPDATE workouts SET
		  status = ?, result_time_seconds = ?, result_distance_km = ?,
		  result_heart_rate = ?, result_feeling = ?, image_file_id = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ?
	`, req.Status, req.ResultTimeSeconds, req.ResultDistanceKm,
		req.ResultHeartRate, req.ResultFeeling, req.ImageFileID, workoutID, studentID)
	if err != nil {
		return 0, "", err
	}
	return coachID, title, nil
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
		if err := rows.Scan(&s.ID, &s.WorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM workout_segments WHERE workout_id = ?", workoutID); err != nil {
		return err
	}
	for i, seg := range segs {
		if _, err := tx.Exec(`
			INSERT INTO workout_segments
			  (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
			   work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *workoutRepository) GetFileUUID(fileID int64) (string, error) {
	var uuid string
	err := r.db.QueryRow("SELECT uuid FROM files WHERE id = ?", fileID).Scan(&uuid)
	return uuid, err
}

func scanWorkoutRows(rows *sql.Rows) ([]models.Workout, error) {
	workouts := []models.Workout{}
	for rows.Next() {
		wo, err := scanWorkout(rows)
		if err != nil {
			return nil, err
		}
		workouts = append(workouts, wo)
	}
	return workouts, rows.Err()
}
```

**Note on `ListCoachWorkouts`:** The query joins `users` for `user_name`, but `scanWorkout` scans 21 columns and then there's an extra scan for `u.name`. This requires two separate scans. To keep it simple, use a modified approach: after `scanWorkout`, scan the extra column. However, `scanWorkout` accepts a `row.Scan` interface — `*sql.Rows` implements it. After calling `scanWorkout(rows)` the position is past all 21 cols, so the extra `rows.Scan(&wo.UserName)` won't work. Fix: scan user_name inside the `ListCoachWorkouts` loop inline instead of calling `scanWorkout`. Rewrite that method's rows loop as:

```go
	for rows.Next() {
		var wo models.Workout
		var coachID sql.NullInt64
		var title, description, workoutType, notes, avgPace, userName sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds, calories sql.NullInt64
		var expectedFields sql.NullString
		var resultDistKm sql.NullFloat64
		var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
		var dueDate sql.NullString

		if err := rows.Scan(
			&wo.ID, &wo.UserID, &coachID,
			&title, &description, &workoutType, &notes, &dueDate,
			&distanceKm, &durationSeconds,
			&expectedFields,
			&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
			&avgPace, &calories, &imageFileID,
			&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
			&userName,
		); err != nil {
			return nil, 0, err
		}
		if coachID.Valid { v := coachID.Int64; wo.CoachID = &v }
		if title.Valid { wo.Title = title.String }
		if description.Valid { wo.Description = description.String }
		if workoutType.Valid { wo.Type = workoutType.String }
		if notes.Valid { wo.Notes = notes.String }
		if dueDate.Valid { wo.DueDate = truncateDate(dueDate.String) }
		if distanceKm.Valid { wo.DistanceKm = distanceKm.Float64 }
		if durationSeconds.Valid { wo.DurationSeconds = int(durationSeconds.Int64) }
		if expectedFields.Valid { wo.ExpectedFields = json.RawMessage(expectedFields.String) }
		if resultDistKm.Valid { v := resultDistKm.Float64; wo.ResultDistanceKm = &v }
		if resultTimeSec.Valid { v := int(resultTimeSec.Int64); wo.ResultTimeSeconds = &v }
		if resultHR.Valid { v := int(resultHR.Int64); wo.ResultHeartRate = &v }
		if resultFeeling.Valid { v := int(resultFeeling.Int64); wo.ResultFeeling = &v }
		if avgPace.Valid { wo.AvgPace = avgPace.String }
		if calories.Valid { wo.Calories = int(calories.Int64) }
		if imageFileID.Valid { v := imageFileID.Int64; wo.ImageFileID = &v }
		if userName.Valid { wo.UserName = userName.String }
		workouts = append(workouts, wo)
	}
```

Also add `"time"` to imports in workout_repository.go if needed (it's not needed since `time` is only used by `models.Workout`'s `CreatedAt`/`UpdatedAt` fields scanned via `sql` driver).

- [ ] **Step 2: Build to check repository compiles**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go build ./repository/...
```

Expected: errors about unused `time` import or missing references from coach_repository.go — fix those in the next task.

- [ ] **Step 3: Commit**

```bash
git add repository/workout_repository.go repository/interfaces.go apperr/codes.go
git commit -m "feat: unified WorkoutRepository with coach-assigned CRUD"
```

---

## Task 6: Repository — coach_repository.go (remove assigned-workout methods)

**Files:**
- Modify: `repository/coach_repository.go`

- [ ] **Step 1: Remove assigned-workout methods from coach_repository.go**

Delete the following methods entirely from `coach_repository.go`:
- `ListAssignedWorkouts`
- `CreateAssignedWorkout`
- `GetAssignedWorkout`
- `UpdateAssignedWorkout`
- `GetAssignedWorkoutStatus`
- `DeleteAssignedWorkout`
- `GetMyAssignedWorkouts`
- `UpdateAssignedWorkoutStatus`
- `FetchSegments`
- `GetFileUUID`

Also remove `ErrStatusConflict` from `coach_repository.go` since it's now needed in `workout_repository.go`. Move this declaration to `workout_repository.go`:

```go
var ErrStatusConflict = errors.New("workout already finalized")
```

Add `"errors"` import to `workout_repository.go` if not already present.

- [ ] **Step 2: Update `GetStudentWorkouts` to query unified `workouts` table**

Replace the current `GetStudentWorkouts` method body to query all workouts for a student (both personal and coach-assigned), returning them ordered by `due_date DESC`. Change `date` column reference to `due_date` and scan all unified columns:

```go
func (r *coachRepository) GetStudentWorkouts(studentID int64) ([]models.Workout, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, coach_id, title, description, type, notes, due_date,
			distance_km, duration_seconds, expected_fields,
			result_distance_km, result_time_seconds, result_heart_rate, result_feeling,
			avg_pace, calories, image_file_id, status, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY due_date DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var coachID sql.NullInt64
		var title, description, workoutType, notes, avgPace sql.NullString
		var distanceKm sql.NullFloat64
		var durationSeconds, calories sql.NullInt64
		var expectedFields sql.NullString
		var resultDistKm sql.NullFloat64
		var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
		var dueDate sql.NullString
		if err := rows.Scan(
			&wo.ID, &wo.UserID, &coachID,
			&title, &description, &workoutType, &notes, &dueDate,
			&distanceKm, &durationSeconds,
			&expectedFields,
			&resultDistKm, &resultTimeSec, &resultHR, &resultFeeling,
			&avgPace, &calories, &imageFileID,
			&wo.Status, &wo.CreatedAt, &wo.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if coachID.Valid { v := coachID.Int64; wo.CoachID = &v }
		if title.Valid { wo.Title = title.String }
		if description.Valid { wo.Description = description.String }
		if workoutType.Valid { wo.Type = workoutType.String }
		if notes.Valid { wo.Notes = notes.String }
		if dueDate.Valid { wo.DueDate = truncateDate(dueDate.String) }
		if distanceKm.Valid { wo.DistanceKm = distanceKm.Float64 }
		if durationSeconds.Valid { wo.DurationSeconds = int(durationSeconds.Int64) }
		if expectedFields.Valid { wo.ExpectedFields = json.RawMessage(expectedFields.String) }
		if resultDistKm.Valid { v := resultDistKm.Float64; wo.ResultDistanceKm = &v }
		if resultTimeSec.Valid { v := int(resultTimeSec.Int64); wo.ResultTimeSeconds = &v }
		if resultHR.Valid { v := int(resultHR.Int64); wo.ResultHeartRate = &v }
		if resultFeeling.Valid { v := int(resultFeeling.Int64); wo.ResultFeeling = &v }
		if avgPace.Valid { wo.AvgPace = avgPace.String }
		if calories.Valid { wo.Calories = int(calories.Int64) }
		if imageFileID.Valid { v := imageFileID.Int64; wo.ImageFileID = &v }
		workouts = append(workouts, wo)
	}
	return workouts, nil
}
```

Add `"encoding/json"` to imports in `coach_repository.go` if not already there.

- [ ] **Step 3: Update `GetDailySummary` to query `workouts` table**

In `coach_repository.go`, find `GetDailySummary`. Update the SQL query to reference the `workouts` table instead of `assigned_workouts`, and use `user_id`/`coach_id` instead of `student_id`/`coach_id`:

Change:
```sql
FROM assigned_workouts aw
JOIN coach_students cs ON cs.coach_id = ? AND cs.student_id = aw.student_id
JOIN users u ON u.id = aw.student_id
WHERE cs.status = 'active' AND aw.due_date = ?
```

To:
```sql
FROM workouts w
JOIN coach_students cs ON cs.coach_id = ? AND cs.student_id = w.user_id
JOIN users u ON u.id = w.user_id
WHERE cs.status = 'active' AND w.coach_id = ? AND w.due_date = ?
```

Pass `coachID` twice in args (for JOIN and WHERE). Update all column references from `aw.*` to `w.*` and `aw.student_id` to `w.user_id`. The `DailySummaryWorkout` struct scan remains the same since the column names in the SELECT are the same. Update segment fetches if they reference `assigned_workout_segments` — replace with `workout_segments` + `workout_id`.

Also remove the call to `r.FetchSegments(item.Workout.ID)` if it existed and replace with a query against `workout_segments`:

```go
// fetch segments inline using workout_segments table
segRows, segErr := r.db.Query(`
	SELECT id, workout_id, order_index, segment_type,
		COALESCE(repetitions,1), COALESCE(value,0), COALESCE(unit,''), COALESCE(intensity,''),
		COALESCE(work_value,0), COALESCE(work_unit,''), COALESCE(work_intensity,''),
		COALESCE(rest_value,0), COALESCE(rest_unit,''), COALESCE(rest_intensity,'')
	FROM workout_segments WHERE workout_id = ? ORDER BY order_index
`, item.Workout.ID)
```

- [ ] **Step 4: Update `GetWeeklyLoad` to query unified `workouts` table**

Find `GetWeeklyLoad` in `coach_repository.go`. Update SQL to query `workouts` instead of both `assigned_workouts` and `workouts`. The unified table has all data. Key changes:
- Remove UNION between `assigned_workouts` and `workouts`
- Query just `workouts WHERE user_id = ?`
- `coach_id IS NULL` means personal workout (was: from personal `workouts` table)
- `coach_id IS NOT NULL` means assigned (was: from `assigned_workouts`)
- Date column is now `due_date` for all rows

The query should compute planned km from `distance_km`, actual km from `result_distance_km`, etc., grouping by week. Personal workouts (`coach_id IS NULL`) always count as completed. A row with `coach_id IS NOT NULL AND status = 'pending'` counts as planned only.

Replace the entire query logic with something like:

```go
cutoff := time.Now().UTC().AddDate(0, 0, -weeks*7).Format("2006-01-02")
rows, err := r.db.Query(`
	SELECT
		DATE_FORMAT(DATE_SUB(due_date, INTERVAL WEEKDAY(due_date) DAY), '%Y-%m-%d') AS week_start,
		SUM(CASE WHEN coach_id IS NOT NULL AND status = 'pending' THEN COALESCE(distance_km, 0) ELSE 0 END) AS planned_km,
		SUM(CASE WHEN status = 'completed' THEN COALESCE(result_distance_km, distance_km, 0) ELSE 0 END) AS actual_km,
		SUM(CASE WHEN coach_id IS NOT NULL AND status = 'pending' THEN COALESCE(duration_seconds, 0) ELSE 0 END) AS planned_seconds,
		SUM(CASE WHEN status = 'completed' THEN COALESCE(result_time_seconds, duration_seconds, 0) ELSE 0 END) AS actual_seconds,
		COUNT(CASE WHEN coach_id IS NOT NULL THEN 1 END) AS sessions_planned,
		COUNT(CASE WHEN status = 'completed' THEN 1 END) AS sessions_completed,
		COUNT(CASE WHEN status = 'skipped' THEN 1 END) AS sessions_skipped,
		MAX(CASE WHEN coach_id IS NULL THEN 1 ELSE 0 END) AS has_personal_workouts
	FROM workouts
	WHERE user_id = ? AND due_date >= ?
	GROUP BY week_start
	ORDER BY week_start ASC
`, studentID, cutoff)
```

Then scan the results into `WeeklyLoadEntry` structs as before.

- [ ] **Step 5: Remove unused imports from coach_repository.go**

Remove `"errors"` import if it was only used for `ErrStatusConflict`. Ensure `"encoding/json"` is imported. Keep `"time"` for cutoff date.

- [ ] **Step 6: Build**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go build ./repository/...
```

Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add repository/coach_repository.go repository/workout_repository.go
git commit -m "feat: migrate coach_repository to unified workouts table"
```

---

## Task 7: Repository — assignment_message_repository.go

**Files:**
- Modify: `repository/assignment_message_repository.go`

- [ ] **Step 1: Update all SQL column references**

Replace every occurrence of:
- `assigned_workouts` → `workouts`
- `assigned_workout_id` → `workout_id` (in SQL queries)
- `assigned_workout_segments` → `workout_segments`
- `student_id` (in context of workouts) → `user_id`

In `GetParticipants`:
```go
func (r *assignmentMessageRepository) GetParticipants(workoutID int64) (int64, int64, string, string, error) {
	var coachID, studentID int64
	var status, title string
	err := r.db.QueryRow(
		"SELECT coach_id, user_id, status, COALESCE(title,'') FROM workouts WHERE id = ?", workoutID,
	).Scan(&coachID, &studentID, &status, &title)
	return coachID, studentID, status, title, err
}
```

In `List`: change `am.assigned_workout_id` → `am.workout_id` and scan into `m.WorkoutID`.

In `Create`: change column `assigned_workout_id` → `workout_id`.

In `MarkRead`: change `assigned_workout_id` → `workout_id`.

Rename `GetAssignedWorkoutDetail` → `GetWorkoutDetail`, update the SQL to query `workouts` (not `assigned_workouts`), update column `student_id` → `user_id`, `coach_id` stays, update `assignment_messages am` join to use `am.workout_id`. Return type changes from `models.AssignedWorkout` to `models.Workout`.

The new `GetWorkoutDetail`:

```go
func (r *assignmentMessageRepository) GetWorkoutDetail(workoutID, userID int64) (models.Workout, error) {
	var wo models.Workout
	var coachID sql.NullInt64
	var title, description, workoutType, notes, avgPace sql.NullString
	var distanceKm sql.NullFloat64
	var durationSeconds, calories sql.NullInt64
	var expectedFields sql.NullString
	var resultDistKm sql.NullFloat64
	var resultTimeSec, resultHR, resultFeeling, imageFileID sql.NullInt64
	var dueDate sql.NullString
	var studentName, coachName string
	var unread int

	err := r.db.QueryRow(`
		SELECT w.id, w.user_id, w.coach_id, w.title, w.description, w.type,
			w.distance_km, w.duration_seconds, w.notes, w.expected_fields,
			w.result_time_seconds, w.result_distance_km, w.result_heart_rate, w.result_feeling,
			w.image_file_id, w.status, w.due_date, w.avg_pace, w.calories,
			w.created_at, w.updated_at,
			us.name AS student_name, uc.name AS coach_name,
			(SELECT COUNT(*) FROM assignment_messages am
				WHERE am.workout_id = w.id AND am.sender_id != ? AND am.is_read = FALSE) AS unread_count
		FROM workouts w
		JOIN users us ON us.id = w.user_id
		JOIN users uc ON uc.id = w.coach_id
		WHERE w.id = ? AND w.coach_id IS NOT NULL AND (w.coach_id = ? OR w.user_id = ?)
	`, userID, workoutID, userID, userID).Scan(
		&wo.ID, &wo.UserID, &coachID,
		&title, &description, &workoutType,
		&distanceKm, &durationSeconds, &notes, &expectedFields,
		&resultTimeSec, &resultDistKm, &resultHR, &resultFeeling,
		&imageFileID, &wo.Status, &dueDate, &avgPace, &calories,
		&wo.CreatedAt, &wo.UpdatedAt,
		&studentName, &coachName, &unread,
	)
	if err != nil {
		return models.Workout{}, err
	}
	if coachID.Valid { v := coachID.Int64; wo.CoachID = &v }
	if title.Valid { wo.Title = title.String }
	if description.Valid { wo.Description = description.String }
	if workoutType.Valid { wo.Type = workoutType.String }
	if notes.Valid { wo.Notes = notes.String }
	if dueDate.Valid { wo.DueDate = truncateDate(dueDate.String) }
	if distanceKm.Valid { wo.DistanceKm = distanceKm.Float64 }
	if durationSeconds.Valid { wo.DurationSeconds = int(durationSeconds.Int64) }
	if expectedFields.Valid { wo.ExpectedFields = json.RawMessage(expectedFields.String) }
	if resultDistKm.Valid { v := resultDistKm.Float64; wo.ResultDistanceKm = &v }
	if resultTimeSec.Valid { v := int(resultTimeSec.Int64); wo.ResultTimeSeconds = &v }
	if resultHR.Valid { v := int(resultHR.Int64); wo.ResultHeartRate = &v }
	if resultFeeling.Valid { v := int(resultFeeling.Int64); wo.ResultFeeling = &v }
	if avgPace.Valid { wo.AvgPace = avgPace.String }
	if calories.Valid { wo.Calories = int(calories.Int64) }
	if imageFileID.Valid { v := imageFileID.Int64; wo.ImageFileID = &v }
	wo.UserName = studentName
	wo.CoachName = coachName
	wo.UnreadMessageCount = unread
	return wo, nil
}
```

In `FetchSegments`: update to query `workout_segments WHERE workout_id = ?`, scan `s.WorkoutID` instead of `s.AssignedWorkoutID`.

- [ ] **Step 2: Build**

```bash
go build ./repository/...
```

Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add repository/assignment_message_repository.go
git commit -m "feat: assignment_message_repository updated for unified workouts table"
```

---

## Task 8: Repository — weekly_template_repository.go

**Files:**
- Modify: `repository/weekly_template_repository.go`

- [ ] **Step 1: Update `Assign()` method**

In the `Assign()` method, replace all references to `assigned_workouts` with `workouts` and `assigned_workout_segments` with `workout_segments`. Change column names: `student_id` → `user_id`, `assigned_workout_id` → `workout_id`.

The force-delete block:
```go
if _, err := tx.Exec(
    `DELETE FROM workouts WHERE user_id = ? AND coach_id = ? AND due_date >= ? AND due_date <= ?`,
    req.StudentID, coachID, weekStart, weekEnd,
); err != nil {
    return nil, nil, err
}
```

The conflict check:
```go
if err := tx.QueryRow(
    `SELECT COUNT(*) FROM workouts WHERE user_id = ? AND coach_id = ? AND due_date = ?`,
    req.StudentID, coachID, dateStr,
).Scan(&exists); err != nil {
```

The INSERT:
```go
res, err := tx.Exec(
    `INSERT INTO workouts
     (user_id, coach_id, title, description, type, distance_km, duration_seconds, notes, due_date, status)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
    req.StudentID, coachID, p.day.Title, p.day.Description, p.day.Type,
    p.day.DistanceKm, p.day.DurationSeconds, p.day.Notes, dateStr,
)
```

The segments INSERT:
```go
if _, err := tx.Exec(
    `INSERT INTO workout_segments
     (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
      work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    awID, i, seg.SegmentType, ...
); err != nil {
```

- [ ] **Step 2: Build**

```bash
go build ./repository/...
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add repository/weekly_template_repository.go
git commit -m "feat: weekly template Assign() targets unified workouts table"
```

---

## Task 9: Services

**Files:**
- Modify: `services/workout_service.go`
- Modify: `services/coach_service.go`

- [ ] **Step 1: Rewrite `services/workout_service.go`**

```go
package services

import (
	"database/sql"
	"errors"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/repository"
)

type WorkoutService struct {
	repo     repository.WorkoutRepository
	notifSvc *NotificationService
	userRepo repository.UserRepository
	coachRepo repository.CoachRepository
}

func NewWorkoutService(repo repository.WorkoutRepository, notifSvc *NotificationService, userRepo repository.UserRepository, coachRepo repository.CoachRepository) *WorkoutService {
	return &WorkoutService{repo: repo, notifSvc: notifSvc, userRepo: userRepo, coachRepo: coachRepo}
}

func withSegments(svc *WorkoutService, wo models.Workout) models.Workout {
	segs, err := svc.repo.GetSegments(wo.ID)
	if err != nil {
		log.Printf("ERROR fetching segments for workout %d: %v", wo.ID, err)
		wo.Segments = []models.WorkoutSegment{}
	} else {
		wo.Segments = segs
	}
	if wo.ImageFileID != nil {
		if uuid, err := svc.repo.GetFileUUID(*wo.ImageFileID); err == nil {
			wo.ImageURL = "/api/files/" + uuid + "/download"
		}
	}
	return wo
}

// --- Personal workout methods ---

func (s *WorkoutService) List(userID int64, startDate, endDate string) ([]models.Workout, error) {
	workouts, err := s.repo.List(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	for i := range workouts {
		workouts[i] = withSegments(s, workouts[i])
	}
	return workouts, nil
}

func (s *WorkoutService) GetByID(id, userID int64) (models.Workout, error) {
	wo, err := s.repo.GetByID(id)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if wo.UserID != userID && (wo.CoachID == nil || *wo.CoachID != userID) {
		return models.Workout{}, ErrForbidden
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
	id, err := s.repo.Create(userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if len(req.Segments) > 0 {
		if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
			return models.Workout{}, err
		}
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
	found, err := s.repo.Update(id, userID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if !found {
		return models.Workout{}, ErrNotFound
	}
	if err := s.repo.ReplaceSegments(id, req.Segments); err != nil {
		return models.Workout{}, err
	}
	wo, err := s.repo.GetByID(id)
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) Delete(id, userID int64) error {
	found, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (s *WorkoutService) UpdateStatus(id, userID int64, req models.UpdateWorkoutStatusRequest) error {
	coachID, workoutTitle, err := s.repo.UpdateStatus(id, userID, req)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if errors.Is(err, repository.ErrStatusConflict) {
		return ErrWorkoutFinished
	}
	if err != nil {
		return err
	}
	if req.Status == "completed" || req.Status == "skipped" {
		studentName, _, _ := s.userRepo.GetNameAndAvatar(userID)
		notifType := "workout_completed"
		title, body := "notif_workout_completed_title", "notif_workout_completed_body"
		if req.Status == "skipped" {
			notifType = "workout_skipped"
			title, body = "notif_workout_skipped_title", "notif_workout_skipped_body"
		}
		meta := map[string]interface{}{
			"workout_id":    id,
			"workout_title": workoutTitle,
			"student_name":  studentName,
		}
		_ = s.notifSvc.Create(coachID, notifType, title, body, meta, nil)
	}
	return nil
}

// --- Coach workout methods ---

func (s *WorkoutService) GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error) {
	workouts, err := s.repo.GetMyWorkouts(studentID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	for i := range workouts {
		workouts[i] = withSegments(s, workouts[i])
	}
	return workouts, nil
}

func (s *WorkoutService) CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error) {
	isCoach, err := s.coachRepo.IsCoach(coachID)
	if err != nil {
		return models.Workout{}, err
	}
	if !isCoach {
		return models.Workout{}, ErrNotCoach
	}
	ok, err := s.coachRepo.IsStudentOf(coachID, req.StudentID)
	if err != nil {
		return models.Workout{}, err
	}
	if !ok {
		return models.Workout{}, ErrForbidden
	}
	wo, err := s.repo.CreateCoachWorkout(coachID, req)
	if err != nil {
		return models.Workout{}, err
	}
	if len(req.Segments) > 0 {
		if err := s.repo.ReplaceSegments(wo.ID, req.Segments); err != nil {
			return models.Workout{}, err
		}
	}
	coachName, _, _ := s.userRepo.GetNameAndAvatar(coachID)
	meta := map[string]interface{}{
		"workout_id":    wo.ID,
		"workout_title": req.Title,
		"coach_name":    coachName,
	}
	_ = s.notifSvc.Create(req.StudentID, "workout_assigned", "notif_workout_assigned_title", "notif_workout_assigned_body", meta, nil)
	return withSegments(s, wo), nil
}

func (s *WorkoutService) ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error) {
	return s.repo.ListCoachWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
}

func (s *WorkoutService) GetCoachWorkout(workoutID, coachID int64) (models.Workout, error) {
	wo, err := s.repo.GetCoachWorkout(workoutID, coachID)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error) {
	status, err := s.repo.GetWorkoutStatus(workoutID, coachID)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if status != "pending" {
		return models.Workout{}, ErrWorkoutFinished
	}
	wo, err := s.repo.UpdateCoachWorkout(workoutID, coachID, req)
	if err == sql.ErrNoRows {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}
	if err := s.repo.ReplaceSegments(workoutID, req.Segments); err != nil {
		return models.Workout{}, err
	}
	return withSegments(s, wo), nil
}

func (s *WorkoutService) DeleteCoachWorkout(workoutID, coachID int64) error {
	status, err := s.repo.GetWorkoutStatus(workoutID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "pending" {
		return ErrWorkoutFinished
	}
	err = s.repo.DeleteCoachWorkout(workoutID, coachID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}
```

- [ ] **Step 2: Update `services/coach_service.go` — remove assigned-workout methods**

Delete these methods from `coach_service.go`:
- `ListAssignedWorkouts`
- `CreateAssignedWorkout`
- `GetAssignedWorkout`
- `UpdateAssignedWorkout`
- `DeleteAssignedWorkout`
- `GetMyAssignedWorkouts`
- `UpdateAssignedWorkoutStatus`

Keep: `ListStudents`, `EndRelationship`, `GetStudentWorkouts`, `GetDailySummary`, `GetStudentLoad`, `GetMyLoad`.

Update `GetStudentWorkouts` — it now returns `[]models.Workout` (already the case) from unified table.

- [ ] **Step 3: Build services**

```bash
go build ./services/...
```

Expected: errors about `WorkoutService` constructor needing new args — fix in `main.go` (Task 11).

- [ ] **Step 4: Commit**

```bash
git add services/
git commit -m "feat: WorkoutService handles personal + coach-assigned; CoachService simplified"
```

---

## Task 10: Handlers — interfaces.go + workout_handler.go + coach_handler.go

**Files:**
- Modify: `handlers/interfaces.go`
- Modify: `handlers/workout_handler.go`
- Modify: `handlers/coach_handler.go`

- [ ] **Step 1: Update `WorkoutServicer` in `handlers/interfaces.go`**

Replace current `WorkoutServicer`:

```go
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
```

Update `CoachServicer` — remove assigned-workout methods. Final:

```go
type CoachServicer interface {
	ListStudents(coachID int64) ([]models.CoachStudentInfo, error)
	EndRelationship(csID, userID int64) error
	GetStudentWorkouts(coachID, studentID int64) ([]models.Workout, error)
	GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error)
	GetStudentLoad(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
	GetMyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}
```

Update `AssignmentMessageServicer`:

```go
type AssignmentMessageServicer interface {
	ListMessages(workoutID, userID int64) ([]models.AssignmentMessage, error)
	SendMessage(workoutID, senderID int64, body string) (models.AssignmentMessage, error)
	MarkRead(workoutID, userID int64) error
	GetWorkoutDetail(workoutID, userID int64) (models.Workout, error)
}
```

- [ ] **Step 2: Rewrite `handlers/workout_handler.go`**

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type WorkoutHandler struct {
	svc WorkoutServicer
}

func NewWorkoutHandler(svc WorkoutServicer) *WorkoutHandler {
	return &WorkoutHandler{svc: svc}
}

// ListWorkouts handles GET /api/workouts
func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	workouts, err := h.svc.List(userID, startDate, endDate)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.ListWorkouts", apperr.WORKOUT_001, "Failed to fetch workouts")
		return
	}
	writeJSON(w, http.StatusOK, workouts)
}

// GetWorkout handles GET /api/workouts/{id}
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
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.GetWorkout", apperr.WORKOUT_002, "Failed to fetch workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// CreateWorkout handles POST /api/workouts
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
	if req.DueDate == "" {
		writeError(w, http.StatusBadRequest, "due_date is required")
		return
	}
	wo, err := h.svc.Create(userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.CreateWorkout", apperr.WORKOUT_003, "Failed to create workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

// UpdateWorkout handles PUT /api/workouts/{id}
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
	wo, err := h.svc.Update(id, userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateWorkout", apperr.WORKOUT_004, "Failed to update workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// DeleteWorkout handles DELETE /api/workouts/{id}
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
	if err := h.svc.Delete(id, userID); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.DeleteWorkout", apperr.WORKOUT_005, "Failed to delete workout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}

// UpdateWorkoutStatus handles PUT /api/workouts/{id}/status
func (h *WorkoutHandler) UpdateWorkoutStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/status")
	id, err := extractID(path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	var req models.UpdateWorkoutStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Status != "completed" && req.Status != "skipped" {
		writeError(w, http.StatusBadRequest, "status must be completed or skipped")
		return
	}
	if req.Status == "completed" && (req.ResultFeeling == nil || *req.ResultFeeling < 1 || *req.ResultFeeling > 10) {
		writeError(w, http.StatusBadRequest, "feeling (1-10) is required when completing a workout")
		return
	}
	if req.ResultDistanceKm != nil && (*req.ResultDistanceKm < 0 || *req.ResultDistanceKm > 1000) {
		writeError(w, http.StatusBadRequest, "result_distance_km must be between 0 and 1000")
		return
	}
	if req.ResultTimeSeconds != nil && (*req.ResultTimeSeconds < 0 || *req.ResultTimeSeconds > 86400*7) {
		writeError(w, http.StatusBadRequest, "result_time_seconds must be between 0 and 604800")
		return
	}
	if req.ResultHeartRate != nil && (*req.ResultHeartRate < 0 || *req.ResultHeartRate > 300) {
		writeError(w, http.StatusBadRequest, "result_heart_rate must be between 0 and 300")
		return
	}
	if err := h.svc.UpdateStatus(id, userID, req); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateWorkoutStatus", apperr.WORKOUT_006, "Failed to update workout status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Status updated", "status": req.Status})
}

// ListCoachWorkouts handles GET /api/coach/workouts
func (h *WorkoutHandler) ListCoachWorkouts(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var studentID *int64
	if s := r.URL.Query().Get("student_id"); s != "" {
		if sid, err := strconv.ParseInt(s, 10, 64); err == nil {
			studentID = &sid
		}
	}
	statusFilter := r.URL.Query().Get("status")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	const maxPageLimit = 100
	limit, offset := 0, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > maxPageLimit {
				n = maxPageLimit
			}
			limit = n
		}
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 && limit > 0 {
			offset = (n - 1) * limit
		}
	}
	workouts, total, err := h.svc.ListCoachWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.ListCoachWorkouts", apperr.WORKOUT_007, "Failed to fetch coach workouts")
		return
	}
	if limit > 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": workouts, "total": total})
	} else {
		writeJSON(w, http.StatusOK, workouts)
	}
}

// CreateCoachWorkout handles POST /api/coach/workouts
func (h *WorkoutHandler) CreateCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateCoachWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	wo, err := h.svc.CreateCoachWorkout(coachID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.CreateCoachWorkout", apperr.WORKOUT_008, "Failed to create coach workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

// GetCoachWorkout handles GET /api/coach/workouts/{id}
func (h *WorkoutHandler) GetCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	wo, err := h.svc.GetCoachWorkout(id, coachID)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.GetCoachWorkout", apperr.WORKOUT_009, "Failed to fetch coach workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// UpdateCoachWorkout handles PUT /api/coach/workouts/{id}
func (h *WorkoutHandler) UpdateCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	var req models.UpdateCoachWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	wo, err := h.svc.UpdateCoachWorkout(id, coachID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateCoachWorkout", apperr.WORKOUT_010, "Failed to update coach workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// DeleteCoachWorkout handles DELETE /api/coach/workouts/{id}
func (h *WorkoutHandler) DeleteCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	if err := h.svc.DeleteCoachWorkout(id, coachID); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.DeleteCoachWorkout", apperr.WORKOUT_011, "Failed to delete coach workout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}
```

- [ ] **Step 3: Update `handlers/coach_handler.go` — remove assigned-workout methods**

Delete these methods:
- `ListAssignedWorkouts`
- `CreateAssignedWorkout`
- `GetAssignedWorkout`
- `UpdateAssignedWorkout`
- `DeleteAssignedWorkout`
- `GetMyAssignedWorkouts`
- `UpdateAssignedWorkoutStatus`

Keep: `ListStudents`, `EndRelationship`, `GetStudentWorkouts`, `GetDailySummary`, `GetStudentLoad`, `GetMyLoad`, `parseWeeksParam`.

Remove unused imports: `"strconv"` (only needed if still used), `"strings"` (check), `"encoding/json"` (if only used by deleted methods). Keep `"time"`.

- [ ] **Step 4: Build handlers**

```bash
go build ./handlers/...
```

Expected: errors about `AssignmentMessageServicer.GetAssignedWorkoutDetail` — fix in assignment_message service next. Also errors about `WorkoutService` constructor in `main.go`.

- [ ] **Step 5: Commit**

```bash
git add handlers/
git commit -m "feat: WorkoutHandler handles all workout endpoints; CoachHandler simplified"
```

---

## Task 11: Services — assignment_message_service.go + main.go + router.go

**Files:**
- Modify: `services/assignment_message_service.go`
- Modify: `main.go`
- Modify: `router/router.go`

- [ ] **Step 1: Update `services/assignment_message_service.go`**

Find `GetAssignedWorkoutDetail` method and rename it to `GetWorkoutDetail`. Change return type to `models.Workout`. Update the call to `r.repo.GetAssignedWorkoutDetail` → `r.repo.GetWorkoutDetail`. Same for any other references.

Also update `ListMessages`, `SendMessage`, `MarkRead` method signatures if they use `awID` (they stay `workoutID` semantically — just rename the parameter name).

- [ ] **Step 2: Update `main.go` — fix WorkoutService constructor**

`NewWorkoutService` now takes 4 args: `(repo, notifSvc, userRepo, coachRepo)`. fx needs to provide all of them. The `*NotificationService` is already provided as a concrete type. The `CoachRepository` is already provided. Just update the annotation:

```go
fx.Annotate(services.NewWorkoutService,
    fx.As(new(handlers.WorkoutServicer))),
```

This stays the same — fx will inject the deps by type automatically since all 4 types are registered.

- [ ] **Step 3: Update `router/router.go`**

Remove:
```go
// Coach assigned workouts routes
mux.HandleFunc("/api/coach/assigned-workouts", ...)
mux.HandleFunc("/api/coach/assigned-workouts/", ...)

// Student assigned workouts routes
mux.HandleFunc("/api/my-assigned-workouts", ...)
mux.HandleFunc("/api/my-assigned-workouts/", ...)
```

Add (using `workout` handler, not `coach` handler):
```go
// Workout status (athlete completes/skips coach-assigned)
mux.HandleFunc("/api/workouts/", func(w http.ResponseWriter, r *http.Request) {
    if strings.HasSuffix(r.URL.Path, "/status") {
        if r.Method == http.MethodPut {
            workout.UpdateWorkoutStatus(w, r)
        } else {
            http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
        }
        return
    }
    switch r.Method {
    case http.MethodGet:
        workout.GetWorkout(w, r)
    case http.MethodPut:
        workout.UpdateWorkout(w, r)
    case http.MethodDelete:
        workout.DeleteWorkout(w, r)
    default:
        http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
    }
})
```

Add coach workouts routes:
```go
// Coach workouts routes (replaces /api/coach/assigned-workouts)
mux.HandleFunc("/api/coach/workouts", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        workout.ListCoachWorkouts(w, r)
    case http.MethodPost:
        workout.CreateCoachWorkout(w, r)
    default:
        http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
    }
})

mux.HandleFunc("/api/coach/workouts/", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        workout.GetCoachWorkout(w, r)
    case http.MethodPut:
        workout.UpdateCoachWorkout(w, r)
    case http.MethodDelete:
        workout.DeleteCoachWorkout(w, r)
    default:
        http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
    }
})
```

Update `router.New` signature — it already takes `workout *handlers.WorkoutHandler`, no change needed.

- [ ] **Step 4: Full build**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go build ./...
```

Expected: clean build.

- [ ] **Step 5: Smoke test**

```bash
cd ~/Desktop/FitReg/FitRegAPI
export $(cat .env | xargs)
go run main.go &
sleep 2
curl -s http://localhost:8080/health
kill %1
```

Expected: `{"status":"ok"}`.

- [ ] **Step 6: Commit**

```bash
git add services/assignment_message_service.go main.go router/router.go
git commit -m "feat: wire unified workout service, update router with new endpoints"
```

---

## Task 12: Frontend — types/index.ts

**Files:**
- Modify: `src/types/index.ts`

- [ ] **Step 1: Replace `AssignedWorkout` with unified `Workout` type and update `Workout`**

Delete the old `Workout` interface (lines 1–17) and the old `AssignedWorkout` interface (lines 53–78). Replace both with a single unified `Workout`:

```typescript
export interface Workout {
  id: number;
  user_id: number;
  coach_id: number | null;
  title: string;
  description: string;
  type: string;
  notes: string;
  due_date: string;
  distance_km: number;
  duration_seconds: number;
  expected_fields: ExpectedField[] | null;
  result_distance_km: number | null;
  result_time_seconds: number | null;
  result_heart_rate: number | null;
  result_feeling: number | null;
  avg_pace: string;
  calories: number;
  image_file_id: number | null;
  image_url: string | null;
  status: 'pending' | 'completed' | 'skipped';
  segments?: WorkoutSegment[];
  unread_message_count?: number;
  coach_name?: string;
  user_name?: string;
  created_at: string;
  updated_at: string;
}
```

Update `WorkoutSegment` — rename `assigned_workout_id` → `workout_id`:

```typescript
export interface WorkoutSegment {
  id?: number;
  workout_id?: number;
  order_index: number;
  segment_type: 'simple' | 'interval';
  repetitions: number;
  value: number;
  unit: 'km' | 'm' | 'min' | 'sec';
  intensity: 'easy' | 'moderate' | 'fast' | 'sprint';
  work_value: number;
  work_unit: 'km' | 'm' | 'min' | 'sec';
  work_intensity: 'easy' | 'moderate' | 'fast' | 'sprint';
  rest_value: number;
  rest_unit: 'km' | 'm' | 'min' | 'sec';
  rest_intensity: 'easy' | 'moderate' | 'fast' | 'sprint';
}
```

Update `DailySummaryItem` — change `assigned_workout` field to `workout`:

```typescript
export interface DailySummaryItem {
  student_id: number;
  student_name: string;
  student_avatar: string | null;
  workout: Workout | null;
}
```

Update `AssignmentMessage` — rename `assigned_workout_id` → `workout_id`:

```typescript
export interface AssignmentMessage {
  id: number;
  workout_id: number;
  sender_id: number;
  sender_name: string;
  sender_avatar: string;
  body: string;
  is_read: boolean;
  created_at: string;
}
```

- [ ] **Step 2: Commit**

```bash
cd ~/Desktop/FitReg/FitRegFE
git add src/types/index.ts
git commit -m "feat: unified Workout type replaces AssignedWorkout in frontend types"
```

---

## Task 13: Frontend — api/workouts.ts + api/coach.ts

**Files:**
- Modify: `src/api/workouts.ts`
- Modify: `src/api/coach.ts`

- [ ] **Step 1: Rewrite `src/api/workouts.ts`**

```typescript
import client from "./client";
import type { Workout, WorkoutSegment } from "../types";

export async function listWorkouts(startDate?: string, endDate?: string): Promise<Workout[]> {
  const params = new URLSearchParams();
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const qs = params.toString();
  const response = await client.get<Workout[]>(`/workouts${qs ? `?${qs}` : ''}`);
  return response.data || [];
}

export async function getWorkout(id: number): Promise<Workout> {
  const response = await client.get<Workout>(`/workouts/${id}`);
  return response.data;
}

export async function createWorkout(
  data: Omit<Workout, "id" | "user_id" | "coach_id" | "workout_id" | "created_at" | "updated_at"> & { segments?: WorkoutSegment[] }
): Promise<Workout> {
  const response = await client.post<Workout>("/workouts", data);
  return response.data;
}

export async function updateWorkout(
  id: number,
  data: Partial<Omit<Workout, "id" | "user_id" | "coach_id" | "created_at" | "updated_at">> & { segments?: WorkoutSegment[] }
): Promise<Workout> {
  const response = await client.put<Workout>(`/workouts/${id}`, data);
  return response.data;
}

export async function deleteWorkout(id: number): Promise<void> {
  await client.delete(`/workouts/${id}`);
}

export async function updateWorkoutStatus(id: number, data: {
  status: 'completed' | 'skipped';
  result_time_seconds?: number | null;
  result_distance_km?: number | null;
  result_heart_rate?: number | null;
  result_feeling?: number | null;
  image_file_id?: number | null;
}): Promise<void> {
  await client.put(`/workouts/${id}/status`, data);
}

// Coach workout functions
export const listCoachWorkouts = (studentId?: number, status?: string, page?: number, limit?: number, startDate?: string, endDate?: string) => {
  const params = new URLSearchParams();
  if (studentId) params.set('student_id', String(studentId));
  if (status) params.set('status', status);
  if (page) params.set('page', String(page));
  if (limit) params.set('limit', String(limit));
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const qs = params.toString();
  return client.get<Workout[] | { data: Workout[]; total: number }>(`/coach/workouts${qs ? `?${qs}` : ''}`);
};

export const createCoachWorkout = (data: {
  student_id: number;
  title: string;
  description: string;
  type: string;
  distance_km: number;
  duration_seconds: number;
  notes: string;
  due_date: string;
  segments?: WorkoutSegment[];
}) => client.post<Workout>('/coach/workouts', data);

export const getCoachWorkout = (id: number) =>
  client.get<Workout>(`/coach/workouts/${id}`);

export const updateCoachWorkout = (id: number, data: {
  title: string;
  description: string;
  type: string;
  distance_km: number;
  duration_seconds: number;
  notes: string;
  due_date: string;
  segments?: WorkoutSegment[];
}) => client.put<Workout>(`/coach/workouts/${id}`, data);

export const deleteCoachWorkout = (id: number) =>
  client.delete(`/coach/workouts/${id}`);
```

- [ ] **Step 2: Update `src/api/coach.ts` — remove assigned-workout functions**

Delete these exported functions:
- `listAssignedWorkouts`
- `createAssignedWorkout`
- `getAssignedWorkout`
- `updateAssignedWorkout`
- `deleteAssignedWorkout`
- `getMyAssignedWorkouts`
- `updateAssignedWorkoutStatus`

Keep: `listStudents`, `addStudent`, `removeStudent`, `getStudentWorkouts`, `getDailySummary`, `getStudentLoad`, `getMyLoad`.

Update import — remove `AssignedWorkout` from type imports. Keep `Student`, `Workout`, `DailySummaryItem`, `WeeklyLoadEntry`.

- [ ] **Step 3: Check TypeScript compiles**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run build 2>&1 | head -50
```

Expected: type errors in pages/components that still reference old API — those are fixed in the next tasks.

- [ ] **Step 4: Commit**

```bash
git add src/api/workouts.ts src/api/coach.ts
git commit -m "feat: update API layer — unified workout endpoints, remove assigned-workout functions"
```

---

## Task 14: Frontend — pages and components

**Files:**
- Delete: `src/pages/MyAssignedWorkouts.tsx`
- Modify: `src/pages/WorkoutForm.tsx`
- Modify: `src/pages/WorkoutList.tsx`
- Modify: `src/pages/AthleteHome.tsx`
- Modify: `src/pages/AssignWorkoutForm.tsx`
- Modify: `src/components/WeeklyStrip.tsx`
- Modify: `src/components/DayModal.tsx`
- Modify: `src/components/MonthCalendar.tsx`

- [ ] **Step 1: Delete `src/pages/MyAssignedWorkouts.tsx`**

```bash
rm ~/Desktop/FitReg/FitRegFE/src/pages/MyAssignedWorkouts.tsx
```

- [ ] **Step 2: Update `src/pages/WorkoutForm.tsx` — remove mandatory segment validation**

Find the validation that requires at least one segment. Remove it. Segments are now optional for personal workouts. If the form has a line like:
```typescript
if (segments.length === 0) { setError('...'); return; }
```
Remove that block.

Also update the POST endpoint — if it calls `createWorkout`, the field `date` becomes `due_date`. Update the request payload field name from `date` to `due_date`.

- [ ] **Step 3: Update `src/pages/WorkoutList.tsx` — unified view**

This page currently shows only personal workouts. Update to:
1. Call `listWorkouts()` (same function, now returns both personal + assigned)
2. Separate workouts: `pending` coach-assigned go to top section, everything else in the table
3. Personal workouts (no `coach_id`) show a "personal" badge instead of coach badge
4. For pending coach-assigned: show Complete/Skip buttons
5. Keep existing pagination for the table section

Key changes:
```typescript
import { listWorkouts, deleteWorkout, updateWorkoutStatus } from "../api/workouts";
import type { Workout } from "../types";

// Split workouts
const pendingAssigned = workouts.filter(w => w.coach_id !== null && w.status === 'pending');
const tableWorkouts = workouts.filter(w => !(w.coach_id !== null && w.status === 'pending'));
```

In the table rows, show `w.coach_id ? w.coach_name : t('personal_workout')` as a source indicator. Only show delete button when `w.coach_id === null`.

- [ ] **Step 4: Update `src/pages/AthleteHome.tsx`**

Replace `getMyAssignedWorkouts` with `listWorkouts` from `api/workouts`:

```typescript
import { listWorkouts } from "../api/workouts";
// ...
const data = await listWorkouts();
setWorkouts(data);
```

Update the type from `AssignedWorkout[]` → `Workout[]`. Pass `workouts` to `WeeklyStrip`.

- [ ] **Step 5: Update `src/pages/AssignWorkoutForm.tsx`**

Replace all references to `/api/coach/assigned-workouts` with `/api/coach/workouts`. Replace `createAssignedWorkout`/`updateAssignedWorkout` imports from `api/coach` with `createCoachWorkout`/`updateCoachWorkout` from `api/workouts`.

Update the type used for `existingWorkout` prop from `AssignedWorkout` to `Workout`.

- [ ] **Step 6: Update `src/components/WeeklyStrip.tsx`**

Replace `AssignedWorkout` type with `Workout`:
```typescript
import type { Workout, FileResponse } from "../types";
import { updateWorkoutStatus } from "../api/workouts";

interface WeeklyStripProps {
  workouts: Workout[];
  onRefresh: () => void;
}
```

Replace `updateAssignedWorkoutStatus` calls with `updateWorkoutStatus`.

In the day cell render: show Complete/Skip buttons only when `workout.coach_id !== null && workout.status === 'pending'`. For personal workouts (`coach_id === null`), show only a "Ver" link.

- [ ] **Step 7: Update `src/components/DayModal.tsx`**

Replace `AssignedWorkout` type with `Workout`:
```typescript
import { updateWorkoutStatus, deleteCoachWorkout } from "../api/workouts";
import type { Workout, FileResponse, WorkoutTemplate } from "../types";

interface DayModalProps {
  date: string;
  workout: Workout | null;
  // ...
}
```

Replace `updateAssignedWorkoutStatus` → `updateWorkoutStatus`. Replace `deleteAssignedWorkout` → `deleteCoachWorkout`.

For the messages link: update `workout.student_id` → `workout.user_id` where applicable. The `AssignedWorkout.student_id` no longer exists; use `workout.user_id` for the student ID in `AssignWorkoutFields`.

In the `AssignWorkoutFields` usage: `studentId={studentId || workout?.user_id || 0}`.

For the complete/skip buttons: only show when `workout.coach_id !== null`. Personal workouts (`coach_id === null`) should NOT show Complete/Skip buttons at all in this modal since they're already completed at creation.

- [ ] **Step 8: Update `src/components/MonthCalendar.tsx`**

Replace `getMyAssignedWorkouts` with `listWorkouts`:
```typescript
import { listWorkouts } from "../api/workouts";
// ...
const res = await listWorkouts(startDate, endDate);
const data: Workout[] = Array.isArray(res) ? res : [];
```

Update type from `AssignedWorkout[]` → `Workout[]`. Pass `Workout[]` to `DayModal`.

- [ ] **Step 9: Build check**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run build 2>&1 | head -80
```

Fix any remaining TypeScript errors. Common ones:
- `workout.date` → `workout.due_date`
- `workout.student_id` → `workout.user_id`
- `workout.assigned_workout` (in DailySummaryItem) → `workout.workout`
- `AssignmentMessage.assigned_workout_id` → `AssignmentMessage.workout_id`

- [ ] **Step 10: Commit**

```bash
git add src/pages/ src/components/
git commit -m "feat: frontend pages and components updated for unified Workout type"
```

---

## Task 15: Frontend — App.tsx, Sidebar.tsx, AssignmentDetail.tsx, Notifications.tsx

**Files:**
- Modify: `src/App.tsx`
- Modify: `src/components/Sidebar.tsx`
- Modify: `src/pages/AssignmentDetail.tsx`
- Modify: `src/pages/Notifications.tsx`

- [ ] **Step 1: Update `src/App.tsx`**

Remove:
```typescript
import MyAssignedWorkouts from "./pages/MyAssignedWorkouts";
// and the route:
<Route path="/my-assignments" element={<ProtectedRoute><MyAssignedWorkouts /></ProtectedRoute>} />
```

- [ ] **Step 2: Update `src/components/Sidebar.tsx`**

Remove the link to `/my-assignments`:
```typescript
// Remove this line:
<Link to="/my-assignments" className={...} onClick={handleNav}>
  ...
</Link>
```

Update the home route active check — remove `!isActive('/my-assignments')` from the condition:
```typescript
className={`sidebar-link ${isActive('/') && !isActive('/workouts') && !isActive('/coaches') ? 'active' : ''}`}
```

- [ ] **Step 3: Update `src/pages/AssignmentDetail.tsx`**

Change `backTo="/my-assignments"` references to `backTo="/workouts"`.

- [ ] **Step 4: Update `src/pages/Notifications.tsx`**

Change navigation targets:
- `/my-assignments` → `/workouts`
- Any reference to `assigned_workout_id` in metadata navigation that goes to `/my-assignments` should go to `/workouts`

- [ ] **Step 5: Final build**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run build
```

Expected: zero TypeScript errors, successful build.

- [ ] **Step 6: Also check for any remaining references to old APIs**

```bash
grep -r "my-assigned-workouts\|MyAssignedWorkouts\|assigned-workouts\|getMyAssignedWorkouts\|updateAssignedWorkoutStatus\|deleteAssignedWorkout\|AssignedWorkout" src/ --include="*.tsx" --include="*.ts" | grep -v node_modules
```

Expected: zero results.

- [ ] **Step 7: Commit**

```bash
git add src/App.tsx src/components/Sidebar.tsx src/pages/AssignmentDetail.tsx src/pages/Notifications.tsx
git commit -m "feat: remove MyAssignedWorkouts page and /my-assignments route"
```

---

## Task 16: CoachDailyView and StudentWorkouts — verify no changes needed

**Files:**
- Read: `src/pages/CoachDailyView.tsx`
- Read: `src/pages/StudentWorkouts.tsx`

- [ ] **Step 1: Check CoachDailyView**

Read `src/pages/CoachDailyView.tsx`. It uses `DailySummaryItem` which has `assigned_workout` field → now renamed to `workout`. Update any `item.assigned_workout` → `item.workout`. Pass `Workout` (not `AssignedWorkout`) to `DayModal`.

- [ ] **Step 2: Check StudentWorkouts**

Read `src/pages/StudentWorkouts.tsx`. The endpoint is already `GET /coach/students/{id}/workouts` — no URL change needed. But the returned type is now `Workout[]` (not `AssignedWorkout[]`). Update the type. The `MonthCalendar` component receives the data — ensure it accepts `Workout[]`.

- [ ] **Step 3: Final build**

```bash
npm run build
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add src/pages/CoachDailyView.tsx src/pages/StudentWorkouts.tsx
git commit -m "feat: update CoachDailyView and StudentWorkouts for unified Workout type"
```

---

## Task 17: End-to-end smoke test + push

- [ ] **Step 1: Start backend**

```bash
cd ~/Desktop/FitReg/FitRegAPI
export $(cat .env | xargs)
go run main.go
```

- [ ] **Step 2: Start frontend**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run dev
```

- [ ] **Step 3: Manual smoke test**

Open browser at `http://localhost:5173`:
1. Log in as athlete → `GET /api/workouts` returns empty array (no 500 errors)
2. Create a personal workout → `POST /api/workouts` with `due_date` field → returns 201 with `coach_id: null` and `status: completed`
3. Log in as coach → assign a workout to student → `POST /api/coach/workouts` → returns 201 with `coach_id` set and `status: pending`
4. Log in as athlete → list workouts → see both personal + assigned
5. Athlete completes the coach workout → `PUT /api/workouts/{id}/status` → status becomes `completed`
6. Check training load chart → shows unified data
7. Check daily summary → `GET /api/coach/daily-summary` → shows coach-assigned workout
8. Check assignment messages → still work at `/assignments/{id}`

- [ ] **Step 4: Push backend to develop**

```bash
cd ~/Desktop/FitReg/FitRegAPI
git push origin develop
```

- [ ] **Step 5: Push frontend to develop**

```bash
cd ~/Desktop/FitReg/FitRegFE
git push origin develop
```

---

## Self-Review

**Spec coverage check:**

| Requirement | Task |
|---|---|
| Drop old tables, create unified `workouts` | Task 1 |
| `workout_segments` with `workout_id` FK | Task 1 |
| `assignment_messages` with `workout_id` FK | Task 1 |
| New unified `Workout` Go model | Task 2 |
| `WorkoutSegment.WorkoutID` field rename | Task 2 |
| Unified `WorkoutRepository` interface | Task 4 |
| `CoachRepository` shrunk | Task 4, 6 |
| `workout_repository.go` full rewrite | Task 5 |
| `coach_repository.go` updated queries | Task 6 |
| `assignment_message_repository.go` updated | Task 7 |
| `weekly_template_repository.go` Assign() updated | Task 8 |
| `WorkoutService` handles both types | Task 9 |
| `CoachService` simplified | Task 9 |
| New workout handler with all endpoints | Task 10 |
| Router: `/api/coach/workouts`, `/api/workouts/{id}/status` | Task 11 |
| Remove `/api/coach/assigned-workouts`, `/api/my-assigned-workouts` | Task 11 |
| Frontend unified `Workout` type | Task 12 |
| Frontend API layer updated | Task 13 |
| `WorkoutForm.tsx` — segments optional | Task 14 |
| `WorkoutList.tsx` — unified view | Task 14 |
| `AthleteHome.tsx` — uses `listWorkouts` | Task 14 |
| `WeeklyStrip.tsx` — checks `coach_id` | Task 14 |
| `DayModal.tsx` — unified type | Task 14 |
| `MonthCalendar.tsx` — unified type | Task 14 |
| Delete `MyAssignedWorkouts.tsx` | Task 15 |
| Remove `/my-assignments` route + nav link | Task 15 |
| `AssignmentDetail.tsx` backTo updated | Task 15 |
| `Notifications.tsx` nav updated | Task 15 |
| `CoachDailyView.tsx` field name updated | Task 16 |
| Template `Assign()` uses new table | Task 8 |

**Potential issues to watch:**
- `GetWeeklyLoad` query in Task 6 must be carefully tested — the aggregation logic changed significantly
- `scanWorkout` helper assumes a fixed column order — verify all callers select in exactly the same order
- `ListCoachWorkouts` requires inline scan (not `scanWorkout` helper) due to extra `user_name` column
- `DailySummaryItem.assigned_workout` renamed to `workout` — must update all FE usages including `CoachDailyView`
