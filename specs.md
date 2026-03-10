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
│   ├── user_handler.go           # Profile CRUD
│   ├── workout_handler.go        # Personal workout CRUD
│   ├── coach_handler.go          # Coach/student/assignment endpoints
│   ├── coach_profile_handler.go  # Coach directory & public profile
│   ├── achievement_handler.go    # Coach achievements CRUD
│   ├── rating_handler.go         # Student ratings for coaches
│   └── admin_handler.go          # Admin panel endpoints
├── models/
│   ├── user.go                   # User model
│   ├── workout.go                # Workout model
│   └── coach.go                  # Coach, AssignedWorkout, Segment, Achievement, Rating models
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
| age | INT | Nullable |
| weight_kg | DECIMAL(5,2) | Nullable |
| language | VARCHAR(5) DEFAULT 'es' | 'es' or 'en' |
| is_coach | BOOLEAN DEFAULT FALSE | Enables coach features |
| is_admin | BOOLEAN DEFAULT FALSE | Enables admin panel access |
| coach_description | TEXT | Coach bio/description |
| coach_public | BOOLEAN DEFAULT FALSE | Visible in coach directory |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### workouts (personal running logs)
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| user_id | BIGINT FK→users | |
| date | DATE | Required |
| distance_km | DECIMAL(6,2) DEFAULT 0 | |
| duration_seconds | INT DEFAULT 0 | |
| avg_pace | VARCHAR(10) | Format: "MM:SS" |
| calories | INT DEFAULT 0 | |
| avg_heart_rate | INT DEFAULT 0 | |
| type | ENUM('easy','tempo','intervals','long_run','race','other') | |
| notes | TEXT | |
| created_at, updated_at | TIMESTAMP | |

### coach_students
| Column | Type | Notes |
|--------|------|-------|
| id | BIGINT PK AUTO_INCREMENT | |
| coach_id | BIGINT FK→users | |
| student_id | BIGINT FK→users | |
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
| GET | /api/me | Get authenticated user profile |
| PUT | /api/me | Update profile (name, sex, age, weight_kg, language, is_coach) |

### Personal Workouts (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/workouts | List user's workouts (desc by date) |
| GET | /api/workouts/{id} | Get workout (owner only) |
| POST | /api/workouts | Create workout → 201 |
| PUT | /api/workouts/{id} | Update workout (owner only) |
| DELETE | /api/workouts/{id} | Delete workout (owner only) |

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
| GET | /api/coach/assigned-workouts | List assignments. Optional `?student_id=X` |
| GET | /api/coach/assigned-workouts/{id} | Get assignment with segments |
| POST | /api/coach/assigned-workouts | Create assignment with segments |
| PUT | /api/coach/assigned-workouts/{id} | Update (blocked if status=completed) |
| DELETE | /api/coach/assigned-workouts/{id} | Delete assignment (cascades segments) |

### Athlete - Assigned Workouts (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/my-assigned-workouts | Get workouts assigned to me (asc by due_date) |
| PUT | /api/my-assigned-workouts/{id}/status | Update status `{ status: "completed"/"skipped" }` |

### Coach Directory (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/coaches | List public coaches with avg rating. Optional `?search=name` |
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

### Admin (JWT + is_admin required)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/admin/stats | Platform metrics (user/coach/student/workout counts) |
| GET | /api/admin/users | List all users with stats |
| PUT | /api/admin/users/{id} | Update user roles (is_coach, is_admin) |
| GET | /api/admin/achievements/pending | List unverified achievements |
| PUT | /api/admin/achievements/{id}/verify | Approve achievement (sets is_verified, verified_by, verified_at) |
| PUT | /api/admin/achievements/{id}/reject | Delete rejected achievement |

## Models

### User
```go
type User struct {
    ID, GoogleID, Email, Name, AvatarURL string
    Sex string  // M, F, other
    Age int
    WeightKg float64
    Language string  // es, en
    IsCoach bool
    IsAdmin bool
    CoachDescription string
    CoachPublic bool
}
```

### Workout
```go
type Workout struct {
    ID, UserID int64
    Date string  // YYYY-MM-DD
    DistanceKm float64
    DurationSeconds int
    AvgPace string  // MM:SS
    Calories, AvgHeartRate int
    Type string  // easy, tempo, intervals, long_run, race, other
    Notes string
}
```

### AssignedWorkout
```go
type AssignedWorkout struct {
    ID, CoachID, StudentID int64
    Title, Description, Type, Notes string
    DistanceKm float64
    DurationSeconds int
    Status string  // pending, completed, skipped
    DueDate string
    StudentName, CoachName string
    Segments []WorkoutSegment
}
```

### WorkoutSegment
```go
type WorkoutSegment struct {
    ID, AssignedWorkoutID int64
    OrderIndex int
    SegmentType string  // simple, interval
    Repetitions int
    // Simple fields
    Value float64; Unit, Intensity string
    // Interval fields
    WorkValue float64; WorkUnit, WorkIntensity string
    RestValue float64; RestUnit, RestIntensity string
}
```

### CoachAchievement
```go
type CoachAchievement struct {
    ID, CoachID int64
    EventName string
    EventDate string
    DistanceKm float64
    ResultTime string
    Position int  // nullable
    IsVerified bool
    VerifiedBy int64  // nullable, admin user ID
    VerifiedAt string  // nullable
}
```

### CoachRating
```go
type CoachRating struct {
    ID, CoachID, StudentID int64
    Rating int  // 1-10
    Comment string
    StudentName, StudentAvatar string  // joined from users
}
```

### CoachListItem
```go
type CoachListItem struct {
    ID int64
    Name, AvatarURL, CoachDescription string
    AvgRating float64
    AchievementCount int
}
```

### CoachPublicProfile
```go
type CoachPublicProfile struct {
    ID int64
    Name, AvatarURL, CoachDescription string
    AvgRating float64
    Achievements []CoachAchievement
    Ratings []CoachRating
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

## Key Implementation Notes

- Nullable DB fields use `sql.Null*` types (NullString, NullInt64, NullFloat64)
- The `exercises` table from migration 001 is unused (replaced by running-specific workouts)
- Coach endpoints verify `is_coach = true` before processing
- Admin endpoints verify `is_admin = true` via `requireAdmin` helper
- Assigned workout edit is blocked if student has marked it `completed`
- Segments are deleted and re-created on workout update (DELETE + INSERT)
- `due_date` represents "training day" (not a deadline)
- Achievement edit is blocked if `is_verified = true`
- Rating upsert uses `ON DUPLICATE KEY UPDATE` on (coach_id, student_id) unique constraint
- Rating endpoint validates student is actually a student of the coach via coach_students table
- Coach directory lists only coaches with `coach_public = true`
- Config reads `PORT` first (Railway injected), falls back to `SERVER_PORT`, then `8080`

## Deployment

- **Platform:** Railway (Go API + MySQL)
- **Branch:** `master` (auto-deploy)
- **Build:** `go build -o main .`
- **Start:** `./main`
- Railway injects `PORT` env var automatically
- MySQL provided as Railway add-on service
