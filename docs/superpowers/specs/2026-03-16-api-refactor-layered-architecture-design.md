# API Refactor: Layered Architecture Design

## Goal

Refactor the FitReg Go API from a single-layer `handlers/` structure into a clean three-layer architecture: **handlers → services → repository**, with UberFX for dependency injection and a `providers/` package for external integrations.

## Context

The current codebase has 13 handler files that mix HTTP parsing, business logic, and raw SQL queries. The refactor separates these concerns to make the code easier to test, extend, and reason about — without changing any external API behavior.

---

## Architecture

### Layers

**`handlers/`** — HTTP boundary only
- Parse request (JSON body, path params, query strings)
- Validate presence and format of required fields
- Call the corresponding service method
- Map the result to an HTTP response (status code + JSON)
- No SQL, no business rules

**`services/`** — Business logic only
- Orchestrate calls to one or more repositories
- Enforce domain rules (e.g., "a student cannot complete a workout assigned to another student")
- Coordinate cross-domain side effects — a service may call another service's methods
- Concrete structs, no interfaces
- No `http.ResponseWriter`, no `*http.Request`

**`repository/`** — Data access only
- All SQL lives here
- Each function performs exactly one DB operation
- Exposes a Go interface per domain (for service-layer mocking in future tests)
- Receives `*sql.DB` as its only dependency

**`providers/`** — External integrations
- `providers/db/` — MySQL connection setup (moved from `database/`). `db.New(cfg *config.Config) (*sql.DB, error)` constructs the DSN and connection pool from config.
- `providers/storage/` — S3 and local file storage (moved from `storage/`). `storage.New(cfg *config.Config) (Storage, error)` replaces the switch block currently in `main.go`. All import paths referencing the old `storage/` package must be updated to `providers/storage/` in Step 0.
- Future providers (payment gateways, weather APIs, etc.) live here as subdirectories

### Dependency chain

```
main.go
  └── fx.App
        ├── providers/db       → *sql.DB
        ├── providers/storage  → Storage
        ├── config.Load        → *config.Config  (passed to constructors that need it)
        ├── repository/*       → depends on *sql.DB
        ├── services/*         → depends on repository interfaces (+ other services where needed)
        ├── handlers/*         → depends on *services
        └── router.New         → fx.Provide, receives all handlers + middleware config
```

---

## Folder Structure

```
FitRegAPI/
├── main.go                              ← fx.New() + startServer
├── handlers/
│   ├── response.go                      ← writeJSON, writeError, logErr (moved here)
│   ├── helpers.go                       ← extractID, truncateDate, and other URL/format utils
│   ├── workout_handler.go
│   ├── coach_handler.go
│   ├── auth_handler.go
│   ├── user_handler.go
│   ├── invitation_handler.go
│   ├── notification_handler.go
│   ├── template_handler.go
│   ├── achievement_handler.go
│   ├── rating_handler.go
│   ├── coach_profile_handler.go
│   ├── assignment_message_handler.go
│   ├── admin_handler.go
│   └── file_handler.go
├── services/
│   ├── workout_service.go
│   ├── coach_service.go
│   ├── auth_service.go
│   ├── user_service.go
│   ├── invitation_service.go
│   ├── notification_service.go
│   ├── template_service.go
│   ├── achievement_service.go
│   ├── rating_service.go
│   ├── coach_profile_service.go
│   ├── assignment_message_service.go
│   ├── admin_service.go
│   └── file_service.go
├── repository/
│   ├── interfaces.go                    ← all repository interfaces
│   ├── workout_repository.go
│   ├── coach_repository.go
│   ├── auth_repository.go
│   ├── user_repository.go
│   ├── invitation_repository.go
│   ├── notification_repository.go
│   ├── template_repository.go
│   ├── achievement_repository.go
│   ├── rating_repository.go
│   ├── coach_profile_repository.go
│   ├── assignment_message_repository.go
│   ├── admin_repository.go
│   └── file_repository.go
├── providers/
│   ├── db/
│   │   └── mysql.go                    ← moved from database/
│   └── storage/
│       ├── storage.go                  ← moved from storage/
│       ├── s3.go
│       └── local.go
├── models/                             ← unchanged
├── middleware/                         ← unchanged
├── router/
│   └── router.go                       ← fx.Provide, receives all handlers as parameters
└── config/                             ← unchanged
```

---

## Shared Handler Utilities

Currently all shared utilities are defined in `workout_handler.go`. During **Step 0** they move to dedicated files in `handlers/`:

| Utility | Destination | Step |
|---------|------------|------|
| `writeJSON` | `handlers/response.go` | 0 |
| `writeError` | `handlers/response.go` | 0 |
| `logErr` | `handlers/response.go` | 0 |
| `extractID` | `handlers/helpers.go` | 0 |
| `truncateDate` | `handlers/helpers.go` | 0 |

### User projection helpers (auth + user domains)

`auth_handler.go` currently defines `userRow`, `userJSON`, `rowToJSON`, and `fillCoachInfo` — unexported helpers for scanning a user DB row and populating coach-related fields. Both `auth_handler.go` and `user_handler.go` use them (same package today).

**In Step 3 (auth migration):** these four helpers move to `services/user_projection.go` as package-level unexported helpers in the `services` package. Both `AuthService` and `UserService` are in the same `services` package, so both can use them without any circular dependency. This file must be created at the start of Step 3, before touching `UserHandler` in Step 4.

### `fetchSegments` helper

`fetchSegments` is currently duplicated in both `coach_handler.go` and `workout_handler.go` (same package, no conflict today). Post-migration, both `CoachRepository` and `WorkoutRepository` need it.

**Decision:** `fetchSegments` is defined once in `coach_handler.go` and shared across the `handlers` package. It moves to `repository/segment_helpers.go` as a package-level unexported function in the `repository` package. Even though the source is the coach file, it is extracted in **Step 1** (workout migration) because `WorkoutRepository` needs it first. `CoachRepository` then reuses it in Step 12.

---

## Dependency Injection with UberFX

### Why UberFX

UberFX is added as an explicit dependency (`go.uber.org/fx`). It brings `go.uber.org/dig` and `go.uber.org/multierr` as transitive deps. The tradeoff — a heavier dependency graph — is accepted in exchange for automatic dependency resolution as the graph grows (13+ constructors today, more in the future). Manual wiring in `main.go` or `router.go` would require updating call sites every time a new dependency is added to any constructor.

Step 0 includes `go get go.uber.org/fx@latest`.

### main.go pattern

```go
func main() {
    fx.New(
        fx.Provide(
            config.Load,
            db.New,          // providers/db
            storage.New,     // providers/storage

            // Repositories
            repository.NewWorkoutRepository,
            repository.NewCoachRepository,
            // ... all 13

            // Services
            services.NewWorkoutService,
            services.NewCoachService,
            // ... all 13

            // Handlers
            handlers.NewWorkoutHandler,
            handlers.NewCoachHandler,
            // ... all 13

            router.New,
        ),
        fx.Invoke(startServer),
    ).Run()
}
```

### Constructor signatures

```go
// Repository — receives *sql.DB, returns interface
func NewWorkoutRepository(db *sql.DB) repository.WorkoutRepository {
    return &workoutRepository{db: db}
}

// Service — receives repository interface(s), returns concrete struct pointer
func NewWorkoutService(repo repository.WorkoutRepository) *WorkoutService {
    return &WorkoutService{repo: repo}
}

// Handler — receives concrete service pointer, returns concrete struct pointer
func NewWorkoutHandler(svc *services.WorkoutService) *WorkoutHandler {
    return &WorkoutHandler{svc: svc}
}
```

### Config values in constructors

`config.Load()` is registered as an FX provider returning `*config.Config`. Constructors that need config values (e.g., `AuthHandler` needs `JWTSecret` and `GoogleClientID`) receive `*config.Config` as a parameter — FX injects it automatically. Config is not split into per-domain structs; the single `*config.Config` struct is passed where needed.

```go
func NewAuthHandler(svc *services.AuthService, cfg *config.Config) *AuthHandler {
    return &AuthHandler{svc: svc, jwtSecret: cfg.JWTSecret, googleClientID: cfg.GoogleClientID}
}
```

### router.New as FX provider

`router.New` is rewritten **in Step 0** — this is the one piece of the existing codebase that must change immediately. Currently it receives `(db *sql.DB, googleClientID, jwtSecret string, store storage.Storage)`. Post-Step 0 it is registered with `fx.Provide` and receives all constructed handler structs. The route registration logic (the `mux.HandleFunc` calls) does not change, only the function signature and how it gets its dependencies.

```go
// router/router.go
func New(
    workout *handlers.WorkoutHandler,
    coach *handlers.CoachHandler,
    auth *handlers.AuthHandler,
    user *handlers.UserHandler,
    invitation *handlers.InvitationHandler,
    notification *handlers.NotificationHandler,
    template *handlers.TemplateHandler,
    achievement *handlers.AchievementHandler,
    rating *handlers.RatingHandler,
    coachProfile *handlers.CoachProfileHandler,
    assignmentMessage *handlers.AssignmentMessageHandler,
    admin *handlers.AdminHandler,
    file *handlers.FileHandler,
    cfg *config.Config,
) *http.ServeMux {
    mux := http.NewServeMux()
    // route registration — identical to today
    return mux
}
```

`startServer` receives `*http.ServeMux` and wraps it with the CORS and Auth middleware, then calls `http.ListenAndServe`.

---

## Cross-Domain: Notifications

Several handlers currently call `h.Notification.CreateNotification(...)` directly — this is a cross-handler coupling that must be resolved before migrating any of those domains.

**Decision:** `NotificationService` becomes a shared service. Any service that needs to emit a notification receives `*services.NotificationService` as a constructor dependency. FX injects it automatically.

```go
func NewCoachService(
    repo repository.CoachRepository,
    notif *services.NotificationService,
) *CoachService {
    return &CoachService{repo: repo, notif: notif}
}
```

Affected services: `CoachService`, `InvitationService`, `UserService`, `AchievementService`, `AdminService`, `AssignmentMessageService`.

**`NotificationService.ResolveAction`** (renamed from `ExecuteAction` on the HTTP handler to avoid confusion) contains cross-domain transaction logic (invitation acceptance, coach approval/rejection). It stays in `NotificationService` but accesses `InvitationRepository` and `CoachRepository` directly — it does NOT call `InvitationService` or `CoachService`. This avoids circular constructor dependencies while keeping the logic in one place.

The `NotificationHandler.ExecuteAction` HTTP method calls `NotificationService.ResolveAction` — the handler name stays `ExecuteAction` (it's an HTTP endpoint), the service method is named `ResolveAction` to make the distinction clear.

```
NotificationService
  ├── NotificationRepository
  ├── InvitationRepository   ← direct repo access for ResolveAction
  └── CoachRepository        ← direct repo access for ResolveAction
```

**Migration constraint:** `notification` must be migrated before any domain that depends on it (`coach`, `invitation`, `user`, `achievement`, `admin`). The migration order table reflects this.

---

## Repository Interfaces

All interfaces live in `repository/interfaces.go`:

```go
package repository

type WorkoutRepository interface {
    Create(userID int64, w models.Workout) (models.Workout, error)
    GetByID(id, userID int64) (models.Workout, error)
    List(userID int64) ([]models.Workout, error)
    Update(id, userID int64, w models.Workout) error
    Delete(id, userID int64) error
}

// one interface per domain, signatures derived from current SQL usage
```

---

## Migration Strategy

Incremental by domain — the project compiles and works after each step.

### Step 0 — Bootstrap (no behavior change)

1. `go get go.uber.org/fx@latest`
2. Create all new folders (`services/`, `repository/`, `providers/`)
3. Move `database/` → `providers/db/` — update `db.New` to accept `*config.Config` and return `(*sql.DB, error)`
4. Move `storage/` → `providers/storage/` — update `storage.New` to accept `*config.Config`, move the provider switch block from `main.go` into it; update all import paths across the codebase
5. Extract `writeJSON`/`writeError`/`logErr` → `handlers/response.go`
6. Extract `extractID`/`truncateDate` → `handlers/helpers.go`
7. Rewrite `router.New` to accept all 13 existing handler structs as parameters + `*config.Config` (route registration logic unchanged, only the signature changes)
8. Wire `main.go` with `fx.New(fx.Provide(...), fx.Invoke(startServer))` — existing handler structs passed as-is, no logic moved yet
9. Verify: project compiles, all routes respond correctly

### Steps 1–13 — Domain by domain

| Step | Domain | Dependency notes |
|------|--------|-----------------|
| 1 | `workout` | Self-contained. Validates the full pattern end-to-end |
| 2 | `file` | Depends only on Storage provider, no cross-service deps |
| 3 | `auth` | Minimal SQL. Move `userRow`/`userJSON`/`rowToJSON`/`fillCoachInfo` to service layer |
| 4 | `user` | Shares user projection helpers with auth (already moved in step 3) |
| 5 | `template` | Simple CRUD, coaches only |
| 6 | `coach_profile` | Mostly reads |
| 7 | `rating` | Depends on coach_profile repo |
| 8 | **`notification`** | **Must come before coach/invitation/user/achievement/admin** — other services depend on NotificationService |
| 9 | `invitation` | Depends on NotificationService (step 8) |
| 10 | `achievement` | Depends on NotificationService (step 8) |
| 11 | `assignment_message` | Depends on NotificationService (step 8) |
| 12 | `coach` | Most complex. Depends on NotificationService + CoachRepository |
| 13 | `admin` | Last — touches nearly all repos |

### Per-domain migration checklist

For each domain:
1. Define interface in `repository/interfaces.go`
2. Create `repository/<domain>_repository.go` — move all SQL from handler
3. Create `services/<domain>_service.go` — move business logic from handler
4. Slim down `handlers/<domain>_handler.go` — HTTP parsing + service call only
5. Update `main.go` FX providers list with the new repo + service constructors
6. Smoke test the affected routes

---

## Error Handling

No changes to the existing error handling pattern. `writeJSON`, `writeError`, and `logErr` are moved to `handlers/response.go` but behave identically.

Services return `(result, error)`. Handlers decide the HTTP status code based on the error. A typed error approach (e.g., `ErrNotFound`, `ErrForbidden`) can be introduced as a separate improvement after the refactor is complete.

---

## What Does NOT Change

- All route paths and HTTP methods
- All request/response JSON shapes
- `models/` package
- `middleware/` package
- `config/` package
- Authentication flow
- Database schema
