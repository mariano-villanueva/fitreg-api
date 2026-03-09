package router

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
)

func New(db *sql.DB, googleClientID, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	ah := handlers.NewAuthHandler(db, googleClientID, jwtSecret)
	uh := handlers.NewUserHandler(db)
	wh := handlers.NewWorkoutHandler(db)
	ch := handlers.NewCoachHandler(db)

	// Auth routes (public)
	mux.HandleFunc("/api/auth/google", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			ah.GoogleLogin(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// User profile routes
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			uh.GetProfile(w, r)
		case http.MethodPut:
			uh.UpdateProfile(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Workout routes
	mux.HandleFunc("/api/workouts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			wh.ListWorkouts(w, r)
		case http.MethodPost:
			wh.CreateWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/workouts/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			wh.GetWorkout(w, r)
		case http.MethodPut:
			wh.UpdateWorkout(w, r)
		case http.MethodDelete:
			wh.DeleteWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach student routes
	mux.HandleFunc("/api/coach/students", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ch.ListStudents(w, r)
		case http.MethodPost:
			ch.AddStudent(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/students/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/workouts") {
			if r.Method == http.MethodGet {
				ch.GetStudentWorkouts(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method == http.MethodDelete {
			ch.RemoveStudent(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach assigned workouts routes
	mux.HandleFunc("/api/coach/assigned-workouts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ch.ListAssignedWorkouts(w, r)
		case http.MethodPost:
			ch.CreateAssignedWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/assigned-workouts/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ch.GetAssignedWorkout(w, r)
		case http.MethodPut:
			ch.UpdateAssignedWorkout(w, r)
		case http.MethodDelete:
			ch.DeleteAssignedWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Student assigned workouts routes
	mux.HandleFunc("/api/my-assigned-workouts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			ch.GetMyAssignedWorkouts(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/my-assigned-workouts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/status") {
			ch.UpdateAssignedWorkoutStatus(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Health check (public)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Apply middleware: CORS -> Auth
	return middleware.CORS(middleware.Auth(jwtSecret)(mux))
}
