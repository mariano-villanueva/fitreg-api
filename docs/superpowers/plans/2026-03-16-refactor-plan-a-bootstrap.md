# API Refactor — Plan A: Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire UberFX into main.go, move database/ and storage/ into providers/, extract shared handler utilities into dedicated files, and update router.New to accept injected handler structs — all without changing any route or response behavior.

**Architecture:** No new behavior is introduced. This is a pure structural change: create the folder scaffold, move/update constructors to match FX conventions, and rewire the startup sequence. Every route must respond identically after this plan is complete.

**Tech Stack:** Go 1.24, go.uber.org/fx (UberFX), database/sql, stdlib HTTP

**Spec:** `docs/superpowers/specs/2026-03-16-api-refactor-layered-architecture-design.md`

---

## Chunk 1: Dependency, providers/, and handler utilities

### Task 1: Add UberFX dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd /Users/marvillanuev/Desktop/FitReg/FitRegAPI
go get go.uber.org/fx@latest
```

Expected: `go.mod` now contains `go.uber.org/fx vX.X.X` and `go.uber.org/dig`, `go.uber.org/multierr` as transitive deps.

- [ ] **Step 2: Verify the module graph compiles**

```bash
go build ./...
```

Expected: exits 0 with no output (nothing uses fx yet, but it must resolve).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add go.uber.org/fx dependency"
```

---

### Task 2: Create providers/db/

**Files:**
- Create: `providers/db/mysql.go`

- [ ] **Step 1: Create the directory**

```bash
mkdir -p providers/db
```

- [ ] **Step 2: Create `providers/db/mysql.go`**

This replaces `database/mysql.go`. The only change is the function signature: `New` accepts `*config.Config` instead of a DSN string, so FX can inject it.

```go
package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fitreg/api/config"
	_ "github.com/go-sql-driver/mysql"
)

// New constructs and pings a *sql.DB using configuration from cfg.
// FX will inject *config.Config automatically.
func New(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
```

- [ ] **Step 3: Verify**

```bash
go build ./providers/db/...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add providers/db/mysql.go
git commit -m "refactor: add providers/db package (replaces database/)"
```

---

### Task 3: Create providers/storage/

**Files:**
- Create: `providers/storage/storage.go`
- Create: `providers/storage/local.go`
- Create: `providers/storage/s3.go`

The three files from `storage/` are moved here verbatim. `storage.go` gets one addition: the `New` constructor that replaces the switch block in `main.go`.

- [ ] **Step 1: Create the directory**

```bash
mkdir -p providers/storage
```

- [ ] **Step 2: Create `providers/storage/storage.go`**

Copy the interface from `storage/storage.go` and add the `New` constructor:

```go
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/fitreg/api/config"
)

// Storage abstracts file storage operations.
// Implementations: S3Storage (production), LocalStorage (development).
type Storage interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// New constructs the appropriate Storage implementation based on cfg.StorageProvider.
// FX will inject *config.Config automatically.
func New(cfg *config.Config) (Storage, error) {
	switch cfg.StorageProvider {
	case "s3":
		s3Store, err := NewS3Storage(context.Background(), S3Config{
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Endpoint:  cfg.S3Endpoint,
		})
		if err != nil {
			return nil, fmt.Errorf("init s3 storage: %w", err)
		}
		return s3Store, nil
	default:
		localStore, err := NewLocalStorage(cfg.LocalStoragePath)
		if err != nil {
			return nil, fmt.Errorf("init local storage: %w", err)
		}
		return localStore, nil
	}
}
```

- [ ] **Step 3: Copy `storage/local.go` → `providers/storage/local.go`**

The file is identical except the directory. Package name (`package storage`) stays the same.

```bash
cp storage/local.go providers/storage/local.go
```

- [ ] **Step 4: Copy `storage/s3.go` → `providers/storage/s3.go`**

```bash
cp storage/s3.go providers/storage/s3.go
```

- [ ] **Step 5: Update the import in `handlers/file_handler.go`**

`file_handler.go` imports `github.com/fitreg/api/storage`. Update it to `github.com/fitreg/api/providers/storage`.

Find the import line:
```go
"github.com/fitreg/api/storage"
```
Replace with:
```go
"github.com/fitreg/api/providers/storage"
```

The package name stays `storage`, so no usage changes needed inside the file.

- [ ] **Step 6: Verify**

```bash
go build ./providers/storage/... && go build ./handlers/...
```

Expected: exits 0. (The old `storage/` and `database/` packages still exist — they will be removed after `main.go` is updated in Task 8.)

- [ ] **Step 7: Commit**

```bash
git add providers/storage/ handlers/file_handler.go
git commit -m "refactor: add providers/storage package (replaces storage/)"
```

---

### Task 4: Extract handlers/response.go

**Files:**
- Create: `handlers/response.go`
- Modify: `handlers/workout_handler.go`

The three functions `writeJSON`, `writeError`, and `logErr` currently live at the top of `workout_handler.go`. They belong to the whole `handlers` package — moving them to a dedicated file makes that clear.

- [ ] **Step 1: Read workout_handler.go to find the exact function bodies**

```bash
head -100 handlers/workout_handler.go
```

Locate `writeJSON`, `writeError`, and `logErr`. They appear after the import block.

- [ ] **Step 2: Create `handlers/response.go`**

Create the file with the three functions cut from `workout_handler.go`. The package and imports are the same package (`handlers`). Typical implementation:

```go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if status >= 500 {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("ERROR %s:%d: %s", file, line, message)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func logErr(context string, err error) {
	if err == nil {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	log.Printf("ERROR %s:%d [%s]: %v", file, line, context, err)
}
```

**IMPORTANT:** Copy the exact function bodies from `workout_handler.go` — do not rewrite them. The above is the expected pattern, but the source of truth is the existing file.

- [ ] **Step 3: Remove the three functions from `handlers/workout_handler.go`**

Delete `writeJSON`, `writeError`, and `logErr` from `workout_handler.go`. Also remove any imports that are now only used by those functions (likely `encoding/json`, `runtime` — verify that `workout_handler.go` still needs them for other reasons before removing).

> **Note:** `workout_handler.go` also contains a method `fetchWorkoutSegments` on `*WorkoutHandler`. This is a receiver method — do **not** confuse it with the package-level function `fetchSegments` in `coach_handler.go`. Leave both untouched in this task; they are not being moved here.

- [ ] **Step 4: Verify — no duplicate symbol errors**

```bash
go build ./handlers/...
```

Expected: exits 0. If you get "already declared", a copy of the function was left in `workout_handler.go`.

- [ ] **Step 5: Commit**

```bash
git add handlers/response.go handlers/workout_handler.go
git commit -m "refactor: extract writeJSON/writeError/logErr to handlers/response.go"
```

---

### Task 5: Extract handlers/helpers.go

**Files:**
- Create: `handlers/helpers.go`
- Modify: `handlers/workout_handler.go`

`extractID` and `truncateDate` are URL/format utilities, not HTTP response helpers. They move to their own file.

- [ ] **Step 1: Read workout_handler.go to find extractID and truncateDate**

```bash
grep -n "func extractID\|func truncateDate" handlers/workout_handler.go
```

Note the line numbers, then read those functions.

- [ ] **Step 2: Create `handlers/helpers.go`**

```go
package handlers

import (
	"strconv"
	"strings"
)
```

Then paste the exact bodies of `extractID` and `truncateDate` from `workout_handler.go`.

- [ ] **Step 3: Remove extractID and truncateDate from workout_handler.go**

Delete the two function definitions. Remove `strconv` and `strings` from `workout_handler.go`'s imports if they are no longer used there (check with `go build`).

- [ ] **Step 4: Verify**

```bash
go build ./handlers/...
```

Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add handlers/helpers.go handlers/workout_handler.go
git commit -m "refactor: extract extractID/truncateDate to handlers/helpers.go"
```

---

## Chunk 2: Constructor updates, router rewrite, and FX wiring

### Task 6: Update NewAuthHandler to accept *config.Config

**Files:**
- Modify: `handlers/auth_handler.go`

Currently `NewAuthHandler(db *sql.DB, googleClientID, jwtSecret string)`. FX can inject `*config.Config` directly — no need for plain string params.

- [ ] **Step 1: Read auth_handler.go to understand the AuthHandler struct**

```bash
head -30 handlers/auth_handler.go
```

Verify the struct fields for GoogleClientID and JWTSecret.

- [ ] **Step 2: Update NewAuthHandler**

Find:
```go
func NewAuthHandler(db *sql.DB, googleClientID, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, GoogleClientID: googleClientID, JWTSecret: jwtSecret}
}
```

Replace with:
```go
func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, GoogleClientID: cfg.GoogleClientID, JWTSecret: cfg.JWTSecret}
}
```

- [ ] **Step 3: Add the config import to auth_handler.go**

Add to imports:
```go
"github.com/fitreg/api/config"
```

- [ ] **Step 4: Verify**

```bash
go build ./handlers/...
```

Expected: exits 0. (router.go still compiles because it hasn't changed yet — it will break in the next task, which is intentional.)

- [ ] **Step 5: Commit**

```bash
git add handlers/auth_handler.go
git commit -m "refactor: NewAuthHandler accepts *config.Config instead of plain strings"
```

---

### Task 7: Rewrite router/router.go

**Files:**
- Modify: `router/router.go`

The router no longer constructs handlers — it receives them already built (injected by FX). The entire route registration block stays identical. Only the function signature and the handler construction lines change.

- [ ] **Step 1: Replace the function signature and remove handler construction**

The new `router.go` receives all 13 handler structs and `*config.Config` as parameters. Remove the `db *sql.DB`, `googleClientID`, `jwtSecret string`, and `store storage.Storage` parameters. Remove the 13 `handlers.New*` constructor calls at the top of the function body.

> **Note — spec vs plan:** The spec diagram shows `router.New` returning `*http.ServeMux`. This plan keeps `http.Handler` as the return type (same as today) and keeps the middleware wrapping inside `router.New`. This is intentional for the bootstrap step — it is the smallest safe change. The return type can be revisited in a later plan if needed.

New file:

```go
package router

import (
	"net/http"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
)

func New(
	ah *handlers.AuthHandler,
	nh *handlers.NotificationHandler,
	uh *handlers.UserHandler,
	wh *handlers.WorkoutHandler,
	ih *handlers.InvitationHandler,
	ch *handlers.CoachHandler,
	cph *handlers.CoachProfileHandler,
	achh *handlers.AchievementHandler,
	rth *handlers.RatingHandler,
	adm *handlers.AdminHandler,
	fh *handlers.FileHandler,
	th *handlers.TemplateHandler,
	amh *handlers.AssignmentMessageHandler,
	cfg *config.Config,
) http.Handler {
	mux := http.NewServeMux()

	// ---- PASTE ALL mux.HandleFunc BLOCKS HERE VERBATIM ----
	// Copy every mux.HandleFunc call from the current router.go.
	// Do not change a single route path or handler call.

	return middleware.CORS(middleware.Auth(cfg.JWTSecret)(mux))
}
```

**IMPORTANT:** The `mux.HandleFunc` section must be copied verbatim from the current `router.go`. Only the function signature, the 13 constructor calls, and the `return` line (replacing `jwtSecret` with `cfg.JWTSecret`) change.

- [ ] **Step 2: Update imports**

Remove: `"database/sql"`, `"github.com/fitreg/api/storage"`
Add: `"github.com/fitreg/api/config"`
Keep: `"net/http"`, `"strings"`, `"github.com/fitreg/api/handlers"`, `"github.com/fitreg/api/middleware"`

- [ ] **Step 3: Verify**

```bash
go build ./router/...
```

Expected: exits 0. `main.go` will fail to compile now (it calls the old `router.New` signature) — that is expected and will be fixed in the next task.

- [ ] **Step 4: Commit**

```bash
git add router/router.go
git commit -m "refactor: router.New accepts injected handler structs for FX wiring"
```

---

### Task 8: Rewrite main.go with UberFX

**Files:**
- Modify: `main.go`

Replace the manual wiring with `fx.New`. The `startServer` function uses FX lifecycle hooks so the server shuts down gracefully when FX receives a signal.

- [ ] **Step 1: Note on handler dependencies**

Six handler constructors already take `*handlers.NotificationHandler` as a second parameter: `NewUserHandler`, `NewInvitationHandler`, `NewCoachHandler`, `NewAchievementHandler`, `NewAdminHandler`, `NewAssignmentMessageHandler`. FX resolves this automatically because `handlers.NewNotificationHandler` is registered and returns `*handlers.NotificationHandler`. No changes are needed to these constructors for this bootstrap — just ensure `handlers.NewNotificationHandler` appears in `fx.Provide` **before** or alongside the others (FX handles ordering regardless).

- [ ] **Step 2: Replace main.go entirely**

```go
package main

import (
	"context"
	"log"
	"net/http"

	"go.uber.org/fx"

	"github.com/fitreg/api/config"
	providerdb "github.com/fitreg/api/providers/db"
	providerstorage "github.com/fitreg/api/providers/storage"
	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/router"
)

func main() {
	fx.New(
		fx.Provide(
			config.Load,
			providerdb.New,
			providerstorage.New,

			// Handlers — order does not matter, FX resolves dependencies automatically
			handlers.NewNotificationHandler,
			handlers.NewAuthHandler,
			handlers.NewUserHandler,
			handlers.NewWorkoutHandler,
			handlers.NewInvitationHandler,
			handlers.NewCoachHandler,
			handlers.NewCoachProfileHandler,
			handlers.NewAchievementHandler,
			handlers.NewRatingHandler,
			handlers.NewAdminHandler,
			handlers.NewFileHandler,
			handlers.NewTemplateHandler,
			handlers.NewAssignmentMessageHandler,

			router.New,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(lc fx.Lifecycle, handler http.Handler, cfg *config.Config) {
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: handler,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("Server starting on :%s", cfg.ServerPort)
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("Server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Server shutting down")
			return server.Shutdown(ctx)
		},
	})
}
```

- [ ] **Step 3: Verify full build**

```bash
go build ./...
```

Expected: exits 0. This is the first time the entire project compiles with the new structure.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "refactor: wire main.go with UberFX (bootstrap complete)"
```

---

### Task 9: Remove old database/ and storage/ packages

**Files:**
- Delete: `database/` (entire directory)
- Delete: `storage/` (entire directory)

Nothing imports these packages anymore. Keeping them would cause confusion.

- [ ] **Step 1: Confirm nothing imports the old packages**

```bash
grep -r "fitreg/api/database\|fitreg/api/storage" --include="*.go" .
```

Expected: no matches. If any appear, fix those imports first.

- [ ] **Step 2: Delete the old directories**

```bash
rm -rf database/ storage/
```

- [ ] **Step 3: Verify**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove deprecated database/ and storage/ packages"
```

---

### Task 10: Smoke test

No code changes. Verify the running server behaves identically to before.

- [ ] **Step 1: Start the server**

```bash
cd /Users/marvillanuev/Desktop/FitReg/FitRegAPI
export $(cat .env | xargs)
go run main.go
```

Expected output (UberFX startup logs + our log):
```
[Fx] PROVIDE    *config.Config <= github.com/fitreg/api/config.Load()
[Fx] PROVIDE    *sql.DB <= github.com/fitreg/api/providers/db.New()
[Fx] PROVIDE    storage.Storage <= github.com/fitreg/api/providers/storage.New()
...
[Fx] RUNNING
Server starting on :8080
```

- [ ] **Step 2: Health check**

```bash
curl -s http://localhost:8080/health
```

Expected: `{"status":"ok"}`

- [ ] **Step 3: Auth route reachable (returns error, not 404)**

```bash
curl -s -X POST http://localhost:8080/api/auth/google \
  -H "Content-Type: application/json" \
  -d '{"token":"invalid"}'
```

Expected: a JSON error response (not a 404, not a 500 from routing). The exact error depends on Google token validation.

- [ ] **Step 4: Protected route returns 401 without token**

```bash
curl -s http://localhost:8080/api/workouts
```

Expected: `{"error":"Unauthorized"}` with HTTP 401.

- [ ] **Step 5: Stop the server (Ctrl+C)**

Expected: FX logs `[Fx] STOP` and then the `OnStop` hook prints `Server shutting down` — graceful shutdown via FX lifecycle hooks.

- [ ] **Step 6: Final commit tag**

```bash
git tag refactor/bootstrap-complete
```

---

## Summary of files changed

| File | Action |
|------|--------|
| `go.mod`, `go.sum` | Add `go.uber.org/fx` |
| `providers/db/mysql.go` | Created — replaces `database/mysql.go` |
| `providers/storage/storage.go` | Created — replaces `storage/storage.go` + adds `New()` |
| `providers/storage/local.go` | Copied from `storage/local.go` |
| `providers/storage/s3.go` | Copied from `storage/s3.go` |
| `handlers/response.go` | Created — `writeJSON`, `writeError`, `logErr` |
| `handlers/helpers.go` | Created — `extractID`, `truncateDate` |
| `handlers/workout_handler.go` | Removed 5 functions now in response.go + helpers.go |
| `handlers/auth_handler.go` | `NewAuthHandler` accepts `*config.Config` |
| `handlers/file_handler.go` | Import path updated to `providers/storage` |
| `router/router.go` | Accepts 13 handler structs + `*config.Config` |
| `main.go` | Replaced with FX wiring + lifecycle hooks |
| `database/` | Deleted |
| `storage/` | Deleted |
