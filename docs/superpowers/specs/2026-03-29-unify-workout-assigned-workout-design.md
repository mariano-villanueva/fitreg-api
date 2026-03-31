# Design: Unify Workout + AssignedWorkout

_Date: 2026-03-29_

## Context

FitReg currently has two separate workout models:
- `workouts` — personal workouts logged by athletes (no coach involved)
- `assigned_workouts` — workouts assigned by a coach to a student

This creates duplication: separate tables, repositories, services, handlers, API endpoints, and frontend pages. The coach cannot see self-logged workouts in the training load charts. The frontend has two distinct workout flows.

**Goal:** Unify both into a single `workouts` table where `coach_id = NULL` means self-assigned (personal). The coach sees all student workouts (assigned + personal) in load charts and student views.

---

## Decision: Option B — New unified `workouts` table

Since there are no production data to preserve, we DROP both old tables and CREATE a clean unified table. Template tables are untouched.

---

## Database Schema

### `workouts` (new unified table)

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `user_id` | BIGINT FK → users NOT NULL | The athlete |
| `coach_id` | BIGINT FK → users NULL | NULL = self-assigned |
| `title` | VARCHAR(255) NULL | Optional for self-assigned |
| `description` | TEXT NULL | |
| `type` | VARCHAR(50) NULL | |
| `notes` | TEXT NULL | |
| `due_date` | DATE NOT NULL | Planned date (= workout date for self-assigned) |
| `distance_km` | DECIMAL(10,2) NULL | Planned distance |
| `duration_seconds` | INT NULL | Planned duration |
| `expected_fields` | JSON NULL | Coach-assigned only |
| `result_distance_km` | DECIMAL(10,2) NULL | Actual result |
| `result_time_seconds` | INT NULL | Actual result |
| `result_heart_rate` | INT NULL | Actual result |
| `result_feeling` | INT NULL | 1–10 |
| `avg_pace` | VARCHAR(10) NULL | e.g. "4:30/km" — available for all |
| `calories` | INT NULL | Available for all |
| `image_file_id` | BIGINT FK → files NULL | |
| `status` | ENUM('pending','completed','skipped') DEFAULT 'completed' | Self-assigned: always 'completed' |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

**Rules:**
- `coach_id = NULL` → self-assigned. Created with `status = 'completed'`, `due_date = workout date`. Fields `result_*`, `avg_pace`, `calories` filled directly at creation.
- `coach_id != NULL` → coach-assigned. Standard pending → completed/skipped flow.

### `workout_segments` (replaces both old segment tables)

Same columns as today, FK `workout_id` → `workouts`.

### `assignment_messages`

FK renamed: `assigned_workout_id` → `workout_id`. No other changes.

### Untouched tables

`workout_templates`, `workout_template_segments`, `weekly_templates`, `weekly_template_days`, `weekly_template_day_segments` — no changes.

---

## Migration SQL

```sql
-- Drop old tables (no data migration)
DROP TABLE IF EXISTS assigned_workout_segments;
DROP TABLE IF EXISTS workout_segments;
DROP TABLE IF EXISTS assignment_messages;
DROP TABLE IF EXISTS assigned_workouts;
DROP TABLE IF EXISTS workouts;

-- New unified workouts table
CREATE TABLE workouts (
  id                 BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id            BIGINT        NOT NULL,
  coach_id           BIGINT        NULL,
  title              VARCHAR(255)  NULL,
  description        TEXT          NULL,
  type               VARCHAR(50)   NULL,
  notes              TEXT          NULL,
  due_date           DATE          NOT NULL,
  distance_km        DECIMAL(10,2) NULL,
  duration_seconds   INT           NULL,
  expected_fields    JSON          NULL,
  result_distance_km DECIMAL(10,2) NULL,
  result_time_seconds INT          NULL,
  result_heart_rate  INT           NULL,
  result_feeling     INT           NULL,
  avg_pace           VARCHAR(10)   NULL,
  calories           INT           NULL,
  image_file_id      BIGINT        NULL,
  status             ENUM('pending','completed','skipped') NOT NULL DEFAULT 'completed',
  created_at         DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id)       REFERENCES users(id),
  FOREIGN KEY (coach_id)      REFERENCES users(id),
  FOREIGN KEY (image_file_id) REFERENCES files(id)
);

-- New unified segments table
CREATE TABLE workout_segments (
  id              BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  workout_id      BIGINT        NOT NULL,
  order_index     INT           NOT NULL DEFAULT 0,
  segment_type    ENUM('simple','interval') NOT NULL DEFAULT 'simple',
  repetitions     INT           NOT NULL DEFAULT 1,
  value           DECIMAL(10,2) NULL,
  unit            VARCHAR(10)   NULL,
  intensity       VARCHAR(20)   NULL,
  work_value      DECIMAL(10,2) NULL,
  work_unit       VARCHAR(10)   NULL,
  work_intensity  VARCHAR(20)   NULL,
  rest_value      DECIMAL(10,2) NULL,
  rest_unit       VARCHAR(10)   NULL,
  rest_intensity  VARCHAR(20)   NULL,
  FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE
);

-- Recreate assignment_messages with updated FK
CREATE TABLE assignment_messages (
  id          BIGINT    NOT NULL AUTO_INCREMENT PRIMARY KEY,
  workout_id  BIGINT    NOT NULL,
  sender_id   BIGINT    NOT NULL,
  body        TEXT      NOT NULL,
  is_read     BOOLEAN   NOT NULL DEFAULT 0,
  created_at  DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (workout_id)  REFERENCES workouts(id) ON DELETE CASCADE,
  FOREIGN KEY (sender_id)   REFERENCES users(id)
);
```

---

## Backend Changes

### Models (`models/workout.go`)

Single `Workout` struct replacing both `Workout` and `AssignedWorkout`:

```go
type Workout struct {
    ID                int64            `json:"id"`
    UserID            int64            `json:"user_id"`
    CoachID           *int64           `json:"coach_id"`           // nil = self-assigned
    Title             string           `json:"title"`
    Description       string           `json:"description"`
    Type              string           `json:"type"`
    Notes             string           `json:"notes"`
    DueDate           string           `json:"due_date"`           // YYYY-MM-DD
    DistanceKm        float64          `json:"distance_km"`
    DurationSeconds   int              `json:"duration_seconds"`
    ExpectedFields    json.RawMessage  `json:"expected_fields"`
    ResultDistanceKm  *float64         `json:"result_distance_km"`
    ResultTimeSeconds *int             `json:"result_time_seconds"`
    ResultHeartRate   *int             `json:"result_heart_rate"`
    ResultFeeling     *int             `json:"result_feeling"`
    AvgPace           string           `json:"avg_pace"`
    Calories          int              `json:"calories"`
    ImageFileID       *int64           `json:"image_file_id"`
    Status            string           `json:"status"`
    Segments          []WorkoutSegment `json:"segments"`
    ImageURL          string           `json:"image_url,omitempty"`
    UnreadMessageCount int             `json:"unread_message_count"`
    CoachName         string           `json:"coach_name,omitempty"`
    UserName          string           `json:"user_name,omitempty"`
    CreatedAt         time.Time        `json:"created_at"`
    UpdatedAt         time.Time        `json:"updated_at"`
}
```

`models/coach.go` loses `AssignedWorkout` struct (replaced by `Workout`).

### Repository (`repository/workout_repository.go`)

Replaces both `workout_repository.go` and the assigned-workout methods in `coach_repository.go`.

Key methods:
- `Create(userID int64, req CreateWorkoutRequest) (Workout, error)` — self-assigned
- `GetByID(id int64) (Workout, error)`
- `ListByUser(userID int64, startDate, endDate string) ([]Workout, error)` — all workouts for athlete
- `Update(id, userID int64, req UpdateWorkoutRequest) (Workout, error)`
- `Delete(id, userID int64) error`
- `UpdateStatus(id, userID int64, req UpdateStatusRequest) error` — complete/skip coach-assigned
- `CreateCoachAssigned(coachID int64, req CreateAssignedRequest) (Workout, error)`
- `ListCoachAssigned(coachID int64, studentID *int64, status, startDate, endDate string) ([]Workout, error)`
- `GetWeeklyLoad(studentID int64, weeks int) ([]WeeklyLoadEntry, error)` — queries unified table
- `GetDailySummary(coachID int64, date string) ([]DailySummaryItem, error)`

### Services

- `workout_service.go` — handles both self-assigned and coach-assigned logic
- `coach_service.go` — removes assigned workout methods (moved to workout_service)

### Handlers

- `workout_handler.go` — handles `/api/workouts` and `/api/coach/workouts`
- `coach_handler.go` — removes assigned workout handlers, keeps students/daily/load

### API Endpoints

**Athlete:**
- `GET  /api/workouts` — list all (personal + assigned), optional `?start_date=&end_date=&status=`
- `POST /api/workouts` — create personal workout
- `GET  /api/workouts/{id}`
- `PUT  /api/workouts/{id}` — only if `coach_id = NULL`
- `DELETE /api/workouts/{id}` — only if `coach_id = NULL`
- `PUT  /api/workouts/{id}/status` — complete/skip, only if `coach_id != NULL`

**Coach:**
- `GET  /api/coach/workouts` — list assigned workouts (optional `?student_id=&status=`)
- `POST /api/coach/workouts` — create assigned workout
- `GET  /api/coach/workouts/{id}`
- `PUT  /api/coach/workouts/{id}`
- `DELETE /api/coach/workouts/{id}`

**Unchanged endpoints (path stays the same, internal queries updated):** `/api/coach/daily-summary`, `/api/coach/students/{id}/workouts`, `/api/coach/students/{id}/load`, `/api/me/load`, `/api/assignment-messages/*`, `/api/assigned-workout-detail/*`, all template endpoints.

Note: `weekly_template_repository.go` currently INSERTs into `assigned_workouts` when assigning a weekly template to a student. This INSERT must be updated to target the new `workouts` table with the unified column names.

Note: `/api/coach/students/{id}/workouts` now returns all workouts for that student (personal + assigned) from the unified table.

---

## Frontend Changes

### Removed
- `pages/MyAssignedWorkouts.tsx` — deleted
- Route `/my-assignments` — removed
- `api/coach.ts`: `getMyAssignedWorkouts` function removed

### Updated pages

**`WorkoutList.tsx`** — unified view of all athlete workouts. Shows pending coach-assigned as cards at top, completed/skipped in table below. Personal workouts appear in the table with a "personal" indicator (no coach badge). Replaces `MyAssignedWorkouts`.

**`WorkoutDetail.tsx`** — already generic. Adds results section and messages section when `coach_id != null`.

**`WorkoutForm.tsx`** — endpoint changes to `POST /api/workouts`. Segment requirement relaxed: segments are optional for self-assigned workouts (the current "at least one segment" validation is removed).

**`AthleteHome.tsx`** — `WeeklyStrip` receives workouts from `GET /api/workouts` (unified). Personal workouts appear in the strip with different styling (no Complete/Skip buttons).

**`StudentWorkouts.tsx`** — endpoint changes to `GET /api/coach/students/{id}/workouts`. Shows all student workouts (personal + assigned).

**`AssignWorkoutForm.tsx`** — endpoint changes to `POST /api/coach/workouts` / `PUT /api/coach/workouts/{id}`.

### Updated components

**`WeeklyStrip.tsx`** — checks `coach_id` to decide whether to show Complete/Skip buttons. Self-assigned workouts show "Ver" only.

**`api/workouts.ts`** — `listWorkouts()` → `GET /api/workouts`. New `updateWorkoutStatus()` replaces `updateAssignedWorkoutStatus`.

**`api/coach.ts`** — `listAssignedWorkouts`, `createAssignedWorkout`, etc. → `/api/coach/workouts` endpoints.

### Navbar

Remove link to `/my-assignments`. Athlete workout nav link (`/workouts`) shows unified list.

---

## What Stays the Same

- All template flows (daily templates, weekly templates, assign weekly template)
- Notification system
- Assignment messages (only FK column rename)
- Coach daily view, training load chart (backend query updates internally)
- Weekly compliance dashboard
- Admin panel
- File upload

---

## Out of Scope

- Pagination changes beyond what already exists
- UI redesign beyond necessary structural changes
- Any new features
