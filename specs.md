# FitReg API — Technical Specification

## Overview

FitReg is a fitness coaching platform where coaches assign and manage workouts for their students. The API is a REST HTTP service built in Go using the standard library, with Uber's `fx` for dependency injection and MySQL as the database.

**Base URL:** `http://localhost:8080`
**Auth:** JWT Bearer token (obtained via Google OAuth)

---

## Architecture

### Layers

```
HTTP Request
    → Middleware (Auth → CORS → Rate Limit)
    → Handler (parse input, call service, write response)
    → Service (business logic, validation)
    → Repository (SQL queries, DB access)
    → MySQL
```

### Dependency Injection

`go.uber.org/fx` wires all components at startup. Each domain (workout, template, coach, etc.) has: `Repository → Service → Handler`.

### File Storage

Supports two backends (via `STORAGE_PROVIDER` env var):
- `local` — filesystem under `LOCAL_STORAGE_PATH` (default `./uploads`)
- `s3` — AWS S3 (or compatible)

Files are tracked in the `files` table. Downloads use a UUID, not the internal ID.

---

## Environment Variables

| Variable | Default | Required |
|----------|---------|----------|
| `DB_HOST` | `localhost` | No |
| `DB_PORT` | `3306` | No |
| `DB_USER` | `root` | No |
| `DB_PASSWORD` | _(empty)_ | No |
| `DB_NAME` | `fitreg` | No |
| `SERVER_PORT` / `PORT` | `8080` | No |
| `GOOGLE_CLIENT_ID` | _(none)_ | **Yes** |
| `JWT_SECRET` | `change-me-in-production` | No |
| `STORAGE_PROVIDER` | `local` | No |
| `LOCAL_STORAGE_PATH` | `./uploads` | No |
| `S3_BUCKET` | _(empty)_ | If S3 |
| `S3_REGION` | `us-east-1` | If S3 |
| `AWS_ACCESS_KEY_ID` | _(empty)_ | If S3 |
| `AWS_SECRET_ACCESS_KEY` | _(empty)_ | If S3 |
| `S3_ENDPOINT` | _(empty)_ | If S3 |
| `ALLOWED_ORIGINS` | _(empty)_ | No |

---

## Middleware

### Authentication (`middleware/auth.go`)

- **Public routes** (no token required): `GET /health`, `POST /api/auth/google`
- **All other routes** require `Authorization: Bearer <jwt>` header
- Token claims must contain `user_id` (float64)
- Returns `401 Unauthorized` on failure

### CORS (`middleware/cors.go`)

- Hardcoded allowed origins: `http://localhost:3000`, `http://localhost:5173`
- Extra origins via `ALLOWED_ORIGINS` env var (comma-separated)
- Allowed methods: `GET, POST, PUT, DELETE, OPTIONS`
- Allowed headers: `Content-Type, Authorization`
- Credentials: enabled

### Rate Limiting (`middleware/rate_limit.go`)

- Applied only to `POST /api/auth/google`
- Limit: 10 requests per IP per minute (sliding window)
- Exceeding limit returns `429 Too Many Requests` with `Retry-After: 60`

---

## Database Schema

### `users`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `google_id` | VARCHAR(255) UNIQUE | Google OAuth sub |
| `email` | VARCHAR(255) | |
| `name` | VARCHAR(255) | |
| `avatar_url` | TEXT | Google-provided avatar |
| `custom_avatar` | MEDIUMTEXT | Base64 custom avatar |
| `sex` | ENUM('M','F','other') | |
| `weight_kg` | DECIMAL(5,2) | |
| `birth_date` | DATE NULL | |
| `height_cm` | INT NULL | |
| `language` | VARCHAR(5) | `es` or `en` |
| `is_coach` | BOOLEAN DEFAULT 0 | |
| `is_admin` | BOOLEAN DEFAULT 0 | |
| `coach_description` | TEXT | Coach bio |
| `coach_public` | BOOLEAN DEFAULT 0 | Visible in directory |
| `coach_locality` | VARCHAR(255) | Coach location |
| `coach_level` | VARCHAR(255) | Experience level |
| `onboarding_completed` | BOOLEAN DEFAULT 0 | |
| `created_at` | TIMESTAMP DEFAULT CURRENT_TIMESTAMP | |
| `updated_at` | TIMESTAMP AUTO UPDATE | |

### `files`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `uuid` | VARCHAR(36) UNIQUE | Public download identifier |
| `user_id` | BIGINT FK → users | Owner |
| `original_name` | VARCHAR(255) | Original filename |
| `content_type` | VARCHAR(100) | MIME type |
| `size_bytes` | BIGINT | |
| `storage_key` | VARCHAR(500) | S3 key or local path |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### `workouts`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `user_id` | BIGINT FK → users | |
| `date` | DATE | YYYY-MM-DD |
| `distance_km` | DECIMAL(6,2) | |
| `duration_seconds` | INT | |
| `avg_pace` | VARCHAR(10) | Display string e.g. `4:30/km` |
| `calories` | INT | |
| `avg_heart_rate` | INT | |
| `feeling` | INT NULL | 1–10 scale |
| `type` | ENUM('easy','tempo','intervals','long_run','race','fartlek','other') | |
| `notes` | TEXT | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### `workout_segments`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `workout_id` | BIGINT FK → workouts | |
| `order_index` | INT | |
| `segment_type` | ENUM('simple','interval') | |
| `repetitions` | INT | |
| `value` | DECIMAL(10,2) | |
| `unit` | VARCHAR(10) | e.g. `km`, `min` |
| `intensity` | VARCHAR(20) | e.g. `easy`, `hard` |
| `work_value` | DECIMAL(10,2) | Interval work portion |
| `work_unit` | VARCHAR(10) | |
| `work_intensity` | VARCHAR(20) | |
| `rest_value` | DECIMAL(10,2) | Interval rest portion |
| `rest_unit` | VARCHAR(10) | |
| `rest_intensity` | VARCHAR(20) | |

### `invitations`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `type` | ENUM('coach_invite','student_request') | Who initiated |
| `sender_id` | BIGINT FK → users | |
| `receiver_id` | BIGINT FK → users | |
| `message` | TEXT NULL | Optional message |
| `status` | ENUM('pending','accepted','rejected','cancelled') | |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

### `coach_students`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `student_id` | BIGINT FK → users | |
| `invitation_id` | BIGINT NULL FK → invitations | |
| `status` | ENUM('active','finished') | |
| `started_at` | DATETIME | |
| `finished_at` | DATETIME NULL | |
| `created_at` | DATETIME | |

### `coach_achievements`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `event_name` | VARCHAR(255) | |
| `event_date` | DATE | YYYY-MM-DD |
| `distance_km` | DECIMAL(6,2) | |
| `result_time` | VARCHAR(10) | HH:MM:SS |
| `position` | INT | |
| `extra_info` | VARCHAR(500) | |
| `image_file_id` | BIGINT NULL FK → files | |
| `is_public` | BOOLEAN DEFAULT 0 | |
| `is_verified` | BOOLEAN DEFAULT 0 | Set by admin |
| `rejection_reason` | VARCHAR(200) | |
| `verified_by` | BIGINT NULL FK → users | Admin |
| `verified_at` | TIMESTAMP NULL | |
| `created_at` | TIMESTAMP | |

### `coach_ratings`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `student_id` | BIGINT FK → users | |
| `rating` | INT | 1–5 |
| `comment` | TEXT | |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |
| UNIQUE | `(coach_id, student_id)` | One rating per student-coach pair |

### `assigned_workouts`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `student_id` | BIGINT FK → users | |
| `title` | VARCHAR(255) | |
| `description` | TEXT | |
| `type` | VARCHAR(50) | |
| `distance_km` | DECIMAL(10,2) NULL | |
| `duration_seconds` | INT NULL | |
| `notes` | TEXT | |
| `expected_fields` | JSON | Fields student should fill |
| `result_time_seconds` | INT NULL | Filled by student |
| `result_distance_km` | DECIMAL(10,2) NULL | Filled by student |
| `result_heart_rate` | INT NULL | Filled by student |
| `result_feeling` | INT NULL | 1–10, filled by student |
| `image_file_id` | BIGINT NULL FK → files | Result photo |
| `status` | ENUM('pending','completed','skipped') | |
| `due_date` | DATE | YYYY-MM-DD |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

### `assigned_workout_segments`

Same columns as `workout_segments`, but with `assigned_workout_id` FK instead of `workout_id`.

### `assignment_messages`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `assigned_workout_id` | BIGINT FK → assigned_workouts | |
| `sender_id` | BIGINT FK → users | |
| `body` | TEXT | |
| `is_read` | BOOLEAN DEFAULT 0 | |
| `created_at` | DATETIME | |

### `notifications`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `user_id` | BIGINT FK → users | Recipient |
| `type` | VARCHAR(50) | |
| `title` | VARCHAR(255) | |
| `body` | TEXT | |
| `metadata` | JSON | Extra data |
| `actions` | JSON NULL | Action buttons |
| `is_read` | BOOLEAN DEFAULT 0 | |
| `created_at` | DATETIME | |

### `notification_preferences`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `user_id` | BIGINT FK → users UNIQUE | |
| `workout_assigned` | BOOLEAN DEFAULT 1 | |
| `workout_completed_or_skipped` | BOOLEAN DEFAULT 1 | |
| `assignment_message` | BOOLEAN DEFAULT 1 | |

### `workout_templates`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `title` | VARCHAR(255) | |
| `description` | TEXT | |
| `type` | VARCHAR(50) | |
| `notes` | TEXT | |
| `expected_fields` | JSON NULL | |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

### `workout_template_segments`

Same columns as `workout_segments`, but with `template_id` FK.

### `weekly_templates`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `coach_id` | BIGINT FK → users | |
| `name` | VARCHAR(255) | |
| `description` | TEXT | |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

### `weekly_template_days`

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGINT PK AUTO_INCREMENT | |
| `weekly_template_id` | BIGINT FK → weekly_templates | |
| `day_of_week` | TINYINT | 0=Monday … 6=Sunday |
| `title` | VARCHAR(255) | |
| `description` | TEXT | |
| `type` | VARCHAR(50) | |
| `distance_km` | DECIMAL(10,2) NULL | |
| `duration_seconds` | INT NULL | |
| `notes` | TEXT | |
| `from_template_id` | BIGINT NULL FK → workout_templates | Source template |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |
| UNIQUE | `(weekly_template_id, day_of_week)` | |

### `weekly_template_day_segments`

Same columns as `workout_segments`, but with `weekly_template_day_id` FK.

---

## Data Models (Go structs)

### Segment (shared shape across all segment tables)

```go
type WorkoutSegment struct {
    ID            int64   `json:"id"`
    OrderIndex    int     `json:"order_index"`
    SegmentType   string  `json:"segment_type"`   // "simple" | "interval"
    Repetitions   int     `json:"repetitions"`
    Value         float64 `json:"value"`
    Unit          string  `json:"unit"`
    Intensity     string  `json:"intensity"`
    WorkValue     float64 `json:"work_value"`
    WorkUnit      string  `json:"work_unit"`
    WorkIntensity string  `json:"work_intensity"`
    RestValue     float64 `json:"rest_value"`
    RestUnit      string  `json:"rest_unit"`
    RestIntensity string  `json:"rest_intensity"`
}
```

### AssignedWorkout

```go
type AssignedWorkout struct {
    ID                 int64            `json:"id"`
    CoachID            int64            `json:"coach_id"`
    StudentID          int64            `json:"student_id"`
    Title              string           `json:"title"`
    Description        string           `json:"description"`
    Type               string           `json:"type"`
    DistanceKm         float64          `json:"distance_km"`
    DurationSeconds    int              `json:"duration_seconds"`
    Notes              string           `json:"notes"`
    ExpectedFields     json.RawMessage  `json:"expected_fields"`
    ResultTimeSeconds  *int             `json:"result_time_seconds"`
    ResultDistanceKm   *float64         `json:"result_distance_km"`
    ResultHeartRate    *int             `json:"result_heart_rate"`
    ResultFeeling      *int             `json:"result_feeling"`
    ImageFileID        *int64           `json:"image_file_id"`
    Status             string           `json:"status"`        // pending|completed|skipped
    DueDate            string           `json:"due_date"`      // YYYY-MM-DD
    StudentName        string           `json:"student_name,omitempty"`
    CoachName          string           `json:"coach_name,omitempty"`
    Segments           []WorkoutSegment `json:"segments"`
    ImageURL           string           `json:"image_url,omitempty"`
    UnreadMessageCount int              `json:"unread_message_count"`
    CreatedAt          time.Time        `json:"created_at"`
    UpdatedAt          time.Time        `json:"updated_at"`
}
```

### WeeklyTemplate

```go
type WeeklyTemplate struct {
    ID          int64               `json:"id"`
    CoachID     int64               `json:"coach_id"`
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Days        []WeeklyTemplateDay `json:"days"`
    DayCount    int                 `json:"day_count,omitempty"`
    CreatedAt   string              `json:"created_at"`
    UpdatedAt   string              `json:"updated_at"`
}

type WeeklyTemplateDay struct {
    ID               int64                   `json:"id"`
    WeeklyTemplateID int64                   `json:"weekly_template_id"`
    DayOfWeek        int                     `json:"day_of_week"` // 0=Mon, 6=Sun
    Title            string                  `json:"title"`
    Description      string                  `json:"description"`
    Type             string                  `json:"type"`
    DistanceKm       float64                 `json:"distance_km"`
    DurationSeconds  int                     `json:"duration_seconds"`
    Notes            string                  `json:"notes"`
    FromTemplateID   *int64                  `json:"from_template_id"`
    Segments         []WeeklyTemplateSegment `json:"segments"`
}

type AssignWeeklyTemplateRequest struct {
    StudentID int64  `json:"student_id"`
    StartDate string `json:"start_date"` // YYYY-MM-DD, must be a Monday
    Force     bool   `json:"force"`      // true = overwrite existing week
}

type AssignConflictResponse struct {
    Error            string   `json:"error"`
    ConflictingDates []string `json:"conflicting_dates"` // YYYY-MM-DD
}
```

---

## API Endpoints

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — returns `{"status":"ok"}` |
| `POST` | `/api/auth/google` | Exchange Google credential for JWT |

#### `POST /api/auth/google`

**Rate limited** (10/min/IP)

**Request:**
```json
{ "credential": "<google_id_token>" }
```

**Response 200:**
```json
{
  "token": "<jwt>",
  "user": { /* UserProfile */ }
}
```

---

### User Profile

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/me` | Get own profile |
| `PUT` | `/api/me` | Update profile fields |
| `POST` | `/api/me/avatar` | Upload custom avatar (multipart/form-data, field: `avatar`) |
| `DELETE` | `/api/me/avatar` | Delete custom avatar |

**UserProfile response fields:**
`id, email, name, avatar_url, custom_avatar (base64), sex, birth_date, age (calculated), weight_kg, height_cm, language, is_coach, is_admin, coach_description, coach_public, onboarding_completed, has_coach, coach_id?, coach_name?, coach_avatar?`

**PUT /api/me request:**
```json
{
  "name": "string",
  "sex": "M|F|other",
  "birth_date": "YYYY-MM-DD",
  "weight_kg": 0.0,
  "height_cm": 0,
  "language": "es|en",
  "onboarding_completed": true
}
```

---

### Coach-Request (student connecting to a coach)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach-request` | Get current request status |
| `POST` | `/api/coach-request` | Request a coach (student initiates) |

---

### Workouts (personal)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/workouts` | List own workouts |
| `POST` | `/api/workouts` | Create workout |
| `GET` | `/api/workouts/{id}` | Get workout |
| `PUT` | `/api/workouts/{id}` | Update workout |
| `DELETE` | `/api/workouts/{id}` | Delete workout |

**CreateWorkoutRequest:**
```json
{
  "date": "YYYY-MM-DD",
  "distance_km": 0.0,
  "duration_seconds": 0,
  "avg_pace": "string",
  "calories": 0,
  "avg_heart_rate": 0,
  "feeling": 0,
  "type": "easy|tempo|intervals|long_run|race|fartlek|other",
  "notes": "string",
  "segments": [ /* SegmentRequest[] */ ]
}
```

**SegmentRequest:**
```json
{
  "segment_type": "simple|interval",
  "repetitions": 1,
  "value": 0.0,
  "unit": "km|min|...",
  "intensity": "easy|moderate|hard|...",
  "work_value": 0.0, "work_unit": "", "work_intensity": "",
  "rest_value": 0.0, "rest_unit": "", "rest_intensity": ""
}
```

---

### Coach — Students

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach/students` | List active students |
| `GET` | `/api/coach/students/{studentId}/workouts` | Get student's assigned workouts |
| `GET` | `/api/coach/daily-summary` | Today's summary for all students |
| `PUT` | `/api/coach-students/{id}/end` | End a coach-student relationship |

**GET /api/coach/daily-summary** response:
```json
[{
  "student_id": 1,
  "student_name": "string",
  "student_avatar": "url|null",
  "assigned_workout": { /* DailySummaryWorkout | null */ }
}]
```

---

### Coach — Assigned Workouts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach/assigned-workouts` | List all assigned workouts (coach view) |
| `POST` | `/api/coach/assigned-workouts` | Create assigned workout |
| `GET` | `/api/coach/assigned-workouts/{id}` | Get single assigned workout |
| `PUT` | `/api/coach/assigned-workouts/{id}` | Update assigned workout |
| `DELETE` | `/api/coach/assigned-workouts/{id}` | Delete assigned workout |

**CreateAssignedWorkoutRequest:**
```json
{
  "student_id": 1,
  "title": "string",
  "description": "string",
  "type": "string",
  "distance_km": 0.0,
  "duration_seconds": 0,
  "notes": "string",
  "expected_fields": ["result_time_seconds", "result_distance_km", "result_heart_rate", "result_feeling"],
  "due_date": "YYYY-MM-DD",
  "segments": [ /* SegmentRequest[] */ ]
}
```

---

### Student — Assigned Workouts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/my-assigned-workouts` | Get own assigned workouts |
| `PUT` | `/api/my-assigned-workouts/{id}/status` | Submit workout result |

**UpdateAssignedWorkoutStatusRequest:**
```json
{
  "status": "completed|skipped",
  "result_time_seconds": 0,
  "result_distance_km": 0.0,
  "result_heart_rate": 0,
  "result_feeling": 0,
  "image_file_id": null
}
```

---

### Coach — Workout Templates (daily templates)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach/templates` | List workout templates |
| `POST` | `/api/coach/templates` | Create template |
| `GET` | `/api/coach/templates/{id}` | Get template |
| `PUT` | `/api/coach/templates/{id}` | Update template |
| `DELETE` | `/api/coach/templates/{id}` | Delete template |

**WorkoutTemplate response:**
```json
{
  "id": 1,
  "coach_id": 1,
  "title": "string",
  "description": "string",
  "type": "string",
  "notes": "string",
  "expected_fields": [],
  "segments": [ /* TemplateSegment[] */ ],
  "created_at": "",
  "updated_at": ""
}
```

---

### Coach — Weekly Templates

Weekly templates are 7-day workout schedules that can be assigned to students. Each day (0=Mon, 6=Sun) can have a workout defined.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach/weekly-templates` | List weekly templates |
| `POST` | `/api/coach/weekly-templates` | Create weekly template |
| `GET` | `/api/coach/weekly-templates/{id}` | Get weekly template (with days and segments) |
| `PUT` | `/api/coach/weekly-templates/{id}` | Update name/description |
| `PUT` | `/api/coach/weekly-templates/{id}/days` | Replace all days (full replacement) |
| `POST` | `/api/coach/weekly-templates/{id}/assign` | Assign to a student |
| `DELETE` | `/api/coach/weekly-templates/{id}` | Delete template |

#### `POST /api/coach/weekly-templates/{id}/assign`

**Request:**
```json
{
  "student_id": 1,
  "start_date": "YYYY-MM-DD",
  "force": false
}
```

- `start_date` must be a **Monday**
- `force: false` — returns 409 if any day in that week already has an assigned workout for that student
- `force: true` — deletes all assigned workouts for the student in that Mon–Sun range, then inserts new ones

**Response 200:**
```json
{ "assigned_workout_ids": [1, 2, 3] }
```

**Response 409 (conflict, force=false):**
```json
{
  "error": "conflict",
  "conflicting_dates": ["2025-03-18", "2025-03-19"]
}
```

---

### Coach Profile

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/api/coach/profile` | Update coach profile fields |

**Request:**
```json
{
  "coach_description": "string",
  "coach_public": true
}
```

---

### Coaches Directory

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coaches` | List public coaches (with optional `?search=...`) |
| `GET` | `/api/coaches/{coachId}` | Get coach public profile |
| `GET` | `/api/coaches/{coachId}/ratings` | Get ratings for a coach |
| `POST` | `/api/coaches/{coachId}/ratings` | Create or update rating (student only) |

**CoachListItem:**
```json
{
  "id": 1, "name": "", "avatar_url": "",
  "coach_description": "", "coach_locality": "", "coach_level": "",
  "avg_rating": 4.5, "rating_count": 10, "verified_achievements": 3
}
```

**UpsertRatingRequest:**
```json
{ "rating": 5, "comment": "string" }
```

---

### Coach Achievements

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/coach/achievements` | List own achievements |
| `POST` | `/api/coach/achievements` | Create achievement |
| `PUT` | `/api/coach/achievements/{id}` | Update achievement |
| `DELETE` | `/api/coach/achievements/{id}` | Delete achievement |
| `PUT` | `/api/coach/achievements/{id}/visibility` | Toggle `is_public` |

**CreateAchievementRequest:**
```json
{
  "event_name": "string",
  "event_date": "YYYY-MM-DD",
  "distance_km": 42.2,
  "result_time": "HH:MM:SS",
  "position": 1,
  "extra_info": "string",
  "image_file_id": null
}
```

---

### Invitations

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/invitations` | List own invitations |
| `POST` | `/api/invitations` | Create invitation |
| `GET` | `/api/invitations/{id}` | Get invitation |
| `DELETE` | `/api/invitations/{id}` | Cancel invitation |
| `PUT` | `/api/invitations/{id}/respond` | Accept or reject |

**CreateInvitationRequest:**
```json
{
  "type": "coach_invite|student_request",
  "receiver_email": "string",
  "receiver_id": 0,
  "message": "string"
}
```

**RespondInvitationRequest:**
```json
{ "action": "accept|reject" }
```

---

### Notifications

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/notifications` | List notifications |
| `GET` | `/api/notifications/unread-count` | Get unread count |
| `PUT` | `/api/notifications/read-all` | Mark all as read |
| `PUT` | `/api/notifications/{id}/read` | Mark one as read |
| `POST` | `/api/notifications/{id}/action` | Execute a notification action |

---

### Notification Preferences

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/notification-preferences` | Get preferences |
| `PUT` | `/api/notification-preferences` | Update preferences |

**Request/Response:**
```json
{
  "workout_assigned": true,
  "workout_completed_or_skipped": true,
  "assignment_message": true
}
```

---

### Files

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/files` | Upload a file (multipart/form-data, field: `file`) |
| `GET` | `/api/files/{uuid}/download` | Download file by UUID |
| `DELETE` | `/api/files/{uuid}` | Delete file |

**Upload response:**
```json
{
  "id": 1,
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "original_name": "photo.jpg",
  "content_type": "image/jpeg",
  "size_bytes": 102400,
  "url": "/api/files/550e8400.../download",
  "created_at": "..."
}
```

---

### Assignment Messages

Messages attached to a specific assigned workout, visible to both the coach and student.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/assignment-messages/{awId}` | List messages for an assigned workout |
| `POST` | `/api/assignment-messages/{awId}` | Send a message |
| `PUT` | `/api/assignment-messages/{awId}/read` | Mark messages as read |
| `GET` | `/api/assigned-workout-detail/{awId}` | Get full workout detail with messages |

**SendMessageRequest:**
```json
{ "body": "string" }
```

**Message response:**
```json
{
  "id": 1,
  "assigned_workout_id": 5,
  "sender_id": 1,
  "sender_name": "Coach Name",
  "sender_avatar": "url",
  "body": "string",
  "is_read": false,
  "created_at": "..."
}
```

---

### Admin

All admin endpoints require `is_admin = true` on the authenticated user.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/stats` | Platform statistics |
| `GET` | `/api/admin/users` | List all users |
| `PUT` | `/api/admin/users/{id}` | Update user (e.g. grant coach/admin) |
| `GET` | `/api/admin/achievements/pending` | List unverified achievements |
| `PUT` | `/api/admin/achievements/{id}/verify` | Verify achievement |
| `PUT` | `/api/admin/achievements/{id}/reject` | Reject achievement with reason |

---

## Error Responses

All errors return JSON:

```json
{ "error": "description" }
```

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| 400 | Bad request / validation error |
| 401 | Missing or invalid JWT |
| 403 | Forbidden (not a coach, not admin, wrong owner) |
| 404 | Resource not found |
| 409 | Conflict (e.g. duplicate, week already has workouts) |
| 429 | Rate limit exceeded |
| 500 | Internal server error |

---

## Authentication Flow

1. Client obtains a Google ID token via Google OAuth 2.0
2. Client calls `POST /api/auth/google` with the token in `{ "credential": "..." }`
3. Server verifies the token against `GOOGLE_CLIENT_ID`
4. Server upserts the user in the `users` table (creates if new)
5. Server returns a JWT (signed with `JWT_SECRET`, contains `user_id`)
6. Client stores the JWT and sends it as `Authorization: Bearer <token>` on all subsequent requests

---

## Roles

| Role | Flag | Capabilities |
|------|------|-------------|
| User | default | Manage own workouts, profile, invitations, notifications |
| Student | has active `coach_students` row | Same as user + view assigned workouts, submit results, send messages |
| Coach | `is_coach = true` | All of user + manage templates, weekly templates, assigned workouts, achievements |
| Admin | `is_admin = true` | All of coach + admin panel |

---

## Segment Structure Reference

Segments represent structured workout intervals. A "simple" segment is a single block (e.g. "run 5km at easy pace"). An "interval" segment repeats a work+rest cycle N times.

```
Simple:  { segment_type: "simple", repetitions: 1, value: 5, unit: "km", intensity: "easy" }
Interval: { segment_type: "interval", repetitions: 6,
            work_value: 1, work_unit: "km", work_intensity: "hard",
            rest_value: 2, rest_unit: "min", rest_intensity: "easy" }
```

The same segment shape is used across:
- `workout_segments` (personal workouts)
- `workout_template_segments` (daily templates)
- `weekly_template_day_segments` (weekly templates)
- `assigned_workout_segments` (assigned workouts)
