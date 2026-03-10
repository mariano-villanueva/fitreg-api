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
	cph := handlers.NewCoachProfileHandler(db)
	achh := handlers.NewAchievementHandler(db)
	rth := handlers.NewRatingHandler(db)
	adm := handlers.NewAdminHandler(db)

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

	// Coach profile routes
	mux.HandleFunc("/api/coach/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			cph.UpdateCoachProfile(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach achievements routes
	mux.HandleFunc("/api/coach/achievements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			achh.ListMyAchievements(w, r)
		case http.MethodPost:
			achh.CreateAchievement(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/achievements/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			achh.UpdateAchievement(w, r)
		case http.MethodDelete:
			achh.DeleteAchievement(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach directory routes
	mux.HandleFunc("/api/coaches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			cph.ListCoaches(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coaches/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ratings") {
			switch r.Method {
			case http.MethodGet:
				rth.GetRatings(w, r)
			case http.MethodPost:
				rth.UpsertRating(w, r)
			default:
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method == http.MethodGet {
			cph.GetCoachProfile(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Admin routes
	mux.HandleFunc("/api/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			adm.GetStats(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			adm.ListUsers(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			adm.UpdateUser(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/achievements/pending", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			adm.PendingAchievements(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/achievements/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if strings.HasSuffix(r.URL.Path, "/verify") {
				adm.VerifyAchievement(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/reject") {
				adm.RejectAchievement(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
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
