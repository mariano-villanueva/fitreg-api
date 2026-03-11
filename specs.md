# FitRegAPI - Technical Specification

## Overview

Go REST API for FitReg, a running workout tracking platform with coach-athlete system. Built with Go standard library (no framework), MySQL, JWT auth, and Google OAuth.

## Project Structure

```
FitRegAPI/
├── main.go                       # Entry point
├── go.mod / go.sum               # Dependencies
├── .env                          # Environment config
├── config/
│   └── config.go                 # Config loading from env vars
├── database/
│   └── mysql.go                  # MySQL connection pool
├── router/
│   └── router.go                 # Route definitions
├── middleware/
│   ├── auth.go                   # JWT auth middleware
│   └── cors.go                   # CORS middleware
├── handlers/
│   ├── auth_handler.go           # Google login
│   ├── user_handler.go           # Profile CRUD + coach request
│   ├── workout_handler.go        # Personal workout CRUD
│   ├── coach_handler.go          # Coach/student/assignment endpoints
│   ├── coach_profile_handler.go  # Coach directory & public profile
│   ├── achievement_handler.go    # Coach achievements CRUD
│   ├── rating_handler.go         # Student ratings for coaches
│   ├── invitation_handler.go     # Coach-student invitation system
│   ├── notification_handler.go   # Notification system with actions
│   └── admin_handler.go          # Admin panel endpoints
├── models/
│   ├── user.go                   # User model
│   ├── workout.go                # Workout model
│   ├── coach.go                  # Coach, AssignedWorkout, Segment, Achievement, Rating models
│   ├── invitation.go             # Invitation model
│   └── notification.go           # Notification, NotificationAction, NotificationPreferences models
└── migrations/
    └── 001_schema.sql            # Consolidated schema (all tables)
```

## Dependencies

```
go 1.21
github.com/go-sql-driver/mysql v1.8.1
github.com/golang-jwt/jwt/v5 v5.3.1
```

## Environment Variables (.env)

| Variable | Default | Description |
|----------|---------|-------------|
| DB_HOST | localhost | MySQL host |
| DB_PORT | 3306 | MySQL port |
| DB_USER | root | MySQL user |
| DB_PASSWORD | root | MySQL password |
| DB_NAME | fitreg | Database name |
| PORT | — | HTTP port (Railway injects this) |
| SERVER_PORT | 8080 | HTTP port fallback |
| GOOGLE_CLIENT_ID | — | Google OAuth client ID |
| JWT_SECRET | — | JWT signing secret |
| ALLOWED_ORIGINS | localhost:3000,localhost:5173 | Comma-separated CORS origins |

Port resolution order: `PORT` → `SERVER_PORT` → `8080` (for Railway compatibility).

Load with: `export $(cat .env | xargs) && go run main.go`

## Database Schema

### users
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| google_id | VARCHAR(255) UNIQUE | Google OAuth ID |
| email | VARCHAR(255) | |
| name | VARCHAR(255) | |
| avatar_url | TEXT | Google profile picture |
| sex | ENUM('M','F','other') | Nullable |
| birth_date | DATE | Nullable |
| age | — | Computed from birth_date in Go (not a DB column) |
| weight_kg | DECIMAL(5,2) | Nullable |
| height_cm | INT | Nullable |
| language | VARCHAR(5) DEFAULT 'es' | 'es' or 'en' |
| is_coach | BOOLEAN DEFAULT FALSE | Enabled by admin approval |
| is_admin | BOOLEAN DEFAULT FALSE | Enables admin panel access |
| coach_description | TEXT | Coach bio/description |
| coach_public | BOOLEAN DEFAULT FALSE | Visible in coach directory |
| coach_locality | VARCHAR(255) | Coach location (set during coach request) |
| coach_level | VARCHAR(255) | Comma-separated training levels (e.g. "beginner,advanced") |
| onboarding_completed | BOOLEAN DEFAULT FALSE | User completed onboarding |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### workouts (personal running logs)
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| user_id | BIGINT FK→users | |
| assigned_workout_id | BIGINT NULL | FK reference to assignment that created this workout |
| date | DATE | Required |
| distance_km | DECIMAL(6,2) DEFAULT 0 | |
| duration_seconds | INT DEFAULT 0 | |
| avg_pace | VARCHAR(10) | Format: "MM:SS" |
| calories | INT DEFAULT 0 | |
| avg_heart_rate | INT DEFAULT 0 | |
| type | ENUM('easy','tempo','intervals','long_run','race','fartlek','other') | |
| notes | TEXT | |
| created_at, updated_at | TIMESTAMP | |

### coach_students
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| coach_id | BIGINT FK→users | |
| student_id | BIGINT FK→users | |
| invitation_id | BIGINT FK→invitations | Link to the invitation that created the relationship |
| status | ENUM('active','ended') DEFAULT 'active' | |
| started_at | TIMESTAMP | |
| ended_at | TIMESTAMP | Nullable |
| created_at | TIMESTAMP | |
| | UNIQUE(coach_id, student_id) | |

### assigned_workouts
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| coach_id | BIGINT FK→users | |
| student_id | BIGINT FK→users | |
| title | VARCHAR(255) | Required |
| description | TEXT | |
| type | ENUM('easy','tempo','intervals','long_run','race','fartlek','other') | |
| distance_km | DECIMAL(6,2) | |
| duration_seconds | INT | |
| notes | TEXT | |
| expected_fields | JSON | Array of expected student data: ['time','distance','heart_rate','feeling'] |
| result_time_seconds | INT | Student result: time |
| result_distance_km | DECIMAL(6,2) | Student result: distance |
| result_heart_rate | INT | Student result: heart rate |
| result_feeling | INT (1-10) | Student result: feeling (always required on completion) |
| status | ENUM('pending','completed','skipped') DEFAULT 'pending' | |
| due_date | DATE | Training day |
| created_at, updated_at | TIMESTAMP | |

### assigned_workout_segments
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| assigned_workout_id | BIGINT FK→assigned_workouts (CASCADE) | |
| order_index | INT DEFAULT 0 | Display order |
| segment_type | ENUM('simple','interval') | |
| repetitions | INT DEFAULT 1 | For intervals |
| value | DECIMAL(8,2) | Simple block value |
| unit | ENUM('km','m','min','sec') | Simple block unit |
| intensity | ENUM('easy','moderate','fast','sprint') | Simple block intensity |
| work_value | DECIMAL(8,2) | Interval work value |
| work_unit | ENUM('km','m','min','sec') | |
| work_intensity | ENUM('easy','moderate','fast','sprint') | |
| rest_value | DECIMAL(8,2) | Interval rest value |
| rest_unit | ENUM('km','m','min','sec') | |
| rest_intensity | ENUM('easy','moderate','fast','sprint') | |
| created_at | TIMESTAMP | |

### invitations
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| type | ENUM('coach_invite','student_request') | |
| sender_id | BIGINT FK→users | |
| receiver_id | BIGINT FK→users | |
| message | TEXT | Optional message |
| status | ENUM('pending','accepted','rejected','cancelled') DEFAULT 'pending' | |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### notifications
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| user_id | BIGINT FK→users | Notification recipient |
| type | VARCHAR(50) | e.g. invitation_received, coach_request, workout_assigned |
| title | VARCHAR(255) | i18n key for title |
| body | TEXT | i18n key for body |
| metadata | JSON | Dynamic data for i18n interpolation |
| actions | JSON | Array of NotificationAction (approve/reject buttons etc.) |
| is_read | BOOLEAN DEFAULT FALSE | |
| created_at | TIMESTAMP | |

### notification_preferences
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| user_id | BIGINT FK→users UNIQUE | |
| workout_assigned | BOOLEAN DEFAULT TRUE | |
| workout_completed_or_skipped | BOOLEAN DEFAULT TRUE | |

### coach_achievements
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| coach_id | BIGINT FK→users | |
| event_name | VARCHAR(255) | Race/event name |
| event_date | DATE | Event date |
| distance_km | DECIMAL(6,2) | Distance |
| result_time | VARCHAR(10) | Format: HH:MM:SS |
| position | INT (nullable) | Finish position |
| is_verified | BOOLEAN DEFAULT FALSE | Admin-validated |
| verified_by | BIGINT FK→users (nullable) | Admin who verified |
| verified_at | TIMESTAMP (nullable) | Verification timestamp |
| created_at | TIMESTAMP | |

### coach_ratings
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| coach_id | BIGINT FK→users | |
| student_id | BIGINT FK→users | |
| rating | INT (1-10) | Score |
| comment | TEXT (nullable) | Optional review text |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |
| | UNIQUE(coach_id, student_id) | One rating per student per coach |

## API Endpoints

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check → `{ status: "ok" }` |
| POST | /api/auth/google | Google login → `{ token, user }` |

### User Profile (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/me | Get authenticated user profile (includes `has_coach` field) |
| PUT | /api/me | Update profile (name, sex, birth_date, weight_kg, height_cm, language). Note: `is_coach` is NOT settable here |

### Coach Request (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coach-request | Check coach request status → `{ status: "none" \| "pending" \| "approved" }` |
| POST | /api/coach-request | Submit coach request `{ locality, level: string[] }` → creates notification for all admins |

Coach activation flow:
1. User submits request with locality and training levels (multiple allowed, e.g. ["beginner","advanced"])
2. Levels stored as comma-separated string in `coach_level` column
3. Notification sent to all admin users with approve/reject actions
4. Admin approves → `is_coach = TRUE, coach_public = TRUE`, all other admin notifications for this request cleared
5. Admin rejects → notifications cleared, user notified

### Personal Workouts (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/workouts | List user's workouts (desc by date) |
| GET | /api/workouts/{id} | Get workout (owner only) |
| POST | /api/workouts | Create workout → 201 |
| PUT | /api/workouts/{id} | Update workout (owner only) |
| DELETE | /api/workouts/{id} | Delete workout (owner only) |

Workouts include `assigned_workout_id` when auto-created from assignment completion.

### Coach - Students (JWT + is_coach required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coach/students | List coach's students |
| POST | /api/coach/students | Add student by email `{ email }` → 201 |
| DELETE | /api/coach/students/{id} | Remove student |
| GET | /api/coach/students/{id}/workouts | View student's personal workouts |

### Coach - Assigned Workouts (JWT + is_coach required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coach/assigned-workouts | List assignments. Params: `?student_id=X&status=pending\|finished&page=N&limit=N`. Returns `{ data, total }` when paginated, or plain array |
| GET | /api/coach/assigned-workouts/{id} | Get assignment with segments |
| POST | /api/coach/assigned-workouts | Create assignment with segments + expected_fields |
| PUT | /api/coach/assigned-workouts/{id} | Update (blocked if status=completed) |
| DELETE | /api/coach/assigned-workouts/{id} | Delete assignment (blocked if completed; cascades segments) |

### Athlete - Assigned Workouts (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/my-assigned-workouts | Get workouts assigned to me (asc by due_date) |
| PUT | /api/my-assigned-workouts/{id}/status | Update status `{ status, result_time_seconds?, result_distance_km?, result_heart_rate?, result_feeling? }`. Feeling is always required for completion. Auto-creates workout record on completion. |

### Invitations (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/invitations | List user's invitations (sent + received) |
| POST | /api/invitations | Create invitation `{ type, receiver_id, message? }` |
| GET | /api/invitations/{id} | Get invitation details |
| PUT | /api/invitations/{id}/respond | Accept/reject `{ action: "accept" \| "reject" }` |
| DELETE | /api/invitations/{id} | Cancel invitation (sender only, must be pending) |

Invitation types: `coach_invite` (coach invites student), `student_request` (student requests coach).
Accepting creates a coach_students record. Notifications sent on create, accept, reject.

### Notifications (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/notifications | List notifications (paginated: `?page=N&limit=N`, default 20) |
| GET | /api/notifications/unread-count | Unread count → `{ count }` |
| PUT | /api/notifications/{id}/read | Mark single notification as read |
| PUT | /api/notifications/read-all | Mark all as read |
| POST | /api/notifications/{id}/action | Execute action `{ action: "approve" \| "reject" \| "accept" }` |
| GET | /api/notification-preferences | Get notification preferences |
| PUT | /api/notification-preferences | Update preferences `{ workout_assigned, workout_completed_or_skipped }` |

Supported action types by notification type:
- `invitation_received`: accept, reject (manages invitation lifecycle)
- `coach_request`: approve, reject (manages coach activation)

### Coach Directory (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coaches | Paginated coach list. Params: `?search=X&locality=X&level=X&sort=rating\|name\|newest\|oldest&page=N&limit=N` → `{ data, total }`. Level filter uses `FIND_IN_SET` to match within comma-separated levels. |
| GET | /api/coaches/{id} | Coach public profile (info + achievements + ratings) |

### Coach Profile (JWT + is_coach required)

| Method | Path | Description |
|--------|------|-------------|
| PUT | /api/coach/profile | Update description & public visibility |

### Coach Achievements (JWT + is_coach required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coach/achievements | List my achievements |
| POST | /api/coach/achievements | Create achievement |
| PUT | /api/coach/achievements/{id} | Edit achievement (blocked if verified) |
| DELETE | /api/coach/achievements/{id} | Delete achievement |

### Coach Ratings (JWT + student of that coach)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/coaches/{id}/ratings | Create/update rating (upsert via ON DUPLICATE KEY UPDATE) |
| GET | /api/coaches/{id}/ratings | View coach ratings |

### Coach-Student Relationship (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| PUT | /api/coach-students/{id}/end | End coaching relationship. Either coach or student can end it. |

### Admin (JWT + is_admin required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/admin/stats | Platform metrics (user/coach/rating/pending achievement counts) |
| GET | /api/admin/users | List all users |
| PUT | /api/admin/users/{id} | Update user roles (is_coach, is_admin) |
| GET | /api/admin/achievements/pending | List unverified achievements |
| PUT | /api/admin/achievements/{id}/verify | Approve achievement (notifies coach) |
| PUT | /api/admin/achievements/{id}/reject | Delete rejected achievement |

## Notification System

Notifications use i18n keys for title and body, with metadata for interpolation. Backend stores keys like `notif_workout_assigned_title`, frontend translates using `t(key, { defaultValue: key, ...metadata })`.

### Notification Types
| Type | Title Key | Body Key | Metadata |
|------|-----------|----------|----------|
| invitation_received | notif_coach_invite_title / notif_student_request_title | notif_coach_invite_body / notif_student_request_body | sender_name, sender_avatar, invitation_id |
| invitation_accepted | notif_invitation_accepted_title | notif_invitation_accepted_body | user_name, invitation_id |
| invitation_rejected | notif_invitation_rejected_title | notif_invitation_rejected_body | user_name, invitation_id |
| relationship_ended | notif_relationship_ended_title | notif_relationship_ended_body | user_name |
| workout_assigned | notif_workout_assigned_title | notif_workout_assigned_body | coach_name, workout_title |
| workout_completed | notif_workout_completed_title | notif_workout_completed_body | student_name, workout_title |
| workout_skipped | notif_workout_skipped_title | notif_workout_skipped_body | student_name, workout_title |
| coach_request | notif_coach_request_title | notif_coach_request_body | requester_id, requester_name, requester_avatar, locality, level |
| coach_request_approved | notif_coach_request_approved_title | notif_coach_request_approved_body | requester_name |
| coach_request_rejected | notif_coach_request_rejected_title | notif_coach_request_rejected_body | requester_name |
| achievement_verified | notif_achievement_verified_title | notif_achievement_verified_body | event_name, achievement_id |

## Models

### User
```go
type User struct {
    ID int64; GoogleID, Email, Name, AvatarURL string
    Sex, BirthDate string; WeightKg float64; HeightCm int
    Language string; IsCoach, IsAdmin bool
    CoachDescription string; CoachPublic bool
    CoachLocality, CoachLevel string  // CoachLevel: comma-separated, e.g. "beginner,advanced"
    OnboardingCompleted bool
    HasCoach bool  // Computed: true if user has active coach_students record as student
}
```

### Workout
```go
type Workout struct {
    ID, UserID int64; AssignedWorkoutID *int64
    Date string; DistanceKm float64; DurationSeconds int
    AvgPace string; Calories, AvgHeartRate int
    Type, Notes string
}
```

### AssignedWorkout
```go
type AssignedWorkout struct {
    ID, CoachID, StudentID int64
    Title, Description, Type, Notes string
    DistanceKm float64; DurationSeconds int
    ExpectedFields json.RawMessage
    ResultTimeSeconds, ResultDistanceKm, ResultHeartRate, ResultFeeling *interface{}
    Status, DueDate string
    StudentName, CoachName string
    Segments []WorkoutSegment
}
```

### Invitation
```go
type Invitation struct {
    ID int64; Type string  // coach_invite, student_request
    SenderID, ReceiverID int64; Message string
    Status string  // pending, accepted, rejected, cancelled
    SenderName, SenderAvatar, ReceiverName, ReceiverAvatar string
}
```

### Notification
```go
type Notification struct {
    ID, UserID int64; Type, Title, Body string
    Metadata json.RawMessage; Actions json.RawMessage
    IsRead bool; CreatedAt time.Time
}
type NotificationAction struct {
    Key, Label, Style string  // style: primary, danger, default
}
```

### CoachListItem
```go
type CoachListItem struct {
    ID int64; Name, AvatarURL, CoachDescription string
    CoachLocality, CoachLevel string
    AvgRating float64; RatingCount, VerifiedCount int
}
```

## Authentication Flow

1. Frontend sends Google ID token to `POST /api/auth/google`
2. Backend verifies via Google's `tokeninfo` endpoint
3. User found or created in DB
4. JWT generated (7-day expiration) with `user_id` and `email` claims
5. All protected endpoints require `Authorization: Bearer <token>` header

## Middleware Chain

Request → CORS → Auth → Handler

- **CORS**: Configurable via `ALLOWED_ORIGINS` env var (comma-separated). Defaults to `localhost:3000` and `localhost:5173`. Methods: GET/POST/PUT/DELETE/OPTIONS
- **Auth**: Skips `/health` and `/api/auth/*`, validates JWT, injects user_id into context

## Database Connection Pool

- Max open connections: 25
- Max idle connections: 5
- Connection lifetime: 5 minutes

## Error Logging

All handlers use centralized error logging:

- **`writeError(w, status, message)`**: Automatically logs 5xx errors with file:line via `runtime.Caller`. 4xx errors are not logged (expected client errors).
- **`logErr(context, err)`**: Logs any non-nil error with file:line and a context string. Used for errors that are handled gracefully but need visibility (e.g., silent DB query failures, discarded scan errors).

Both are defined in `workout_handler.go` and shared across all handlers. Format: `ERROR [file:line] context: error`.

## Key Implementation Notes

- Nullable DB fields use `sql.Null*` types (NullString, NullInt64, NullFloat64)
- Coach mode requires admin approval — users submit request via `POST /api/coach-request` with locality and level(s)
- Coach request sends `level` as a JSON array of strings; stored as comma-separated in DB
- `is_coach` is NOT modifiable via profile update endpoint
- Coach request creates notifications for ALL admin users; first admin to act resolves all
- Completing an assignment auto-creates a workout record with `assigned_workout_id` FK
- `result_feeling` (1-10) is always required when completing an assignment
- Assigned workout listing supports `?status=pending|finished` filter and `?page=N&limit=N` pagination
- Coach directory is paginated with search (name + description), locality, level (uses `FIND_IN_SET` for comma-separated matching), and sort (rating/name/newest/oldest) filters
- `has_coach` is computed at login and profile fetch by checking `coach_students` for an active record as student
- Approving a coach request sets both `is_coach = TRUE` and `coach_public = TRUE`
- Notification titles and bodies are i18n keys; metadata provides interpolation values
- Notification actions (approve/reject) are resolved inline via `POST /api/notifications/{id}/action`
- Invitation system: coach_invite and student_request types, with transactional accept using `SELECT FOR UPDATE`
- MaxCoachesPerStudent constant limits concurrent coaching relationships
- `truncateDate()` helper for MySQL DATE with `parseTime=true`
- Segments are deleted and re-created on workout update (DELETE + INSERT)
- `due_date` represents "training day" (not a deadline)
- Achievement edit is blocked if `is_verified = true`
- Rating upsert uses `ON DUPLICATE KEY UPDATE` on (coach_id, student_id) unique constraint

## Deployment

- **Platform:** Railway (Go API + MySQL)
- **Branch:** `master` (auto-deploy)
- **Build:** `go build -o main .`
- **Start:** `./main`
- Railway injects `PORT` env var automatically
- MySQL provided as Railway add-on service
