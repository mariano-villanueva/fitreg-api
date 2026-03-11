package router

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/storage"
)

func New(db *sql.DB, googleClientID, jwtSecret string, store storage.Storage) http.Handler {
	mux := http.NewServeMux()

	ah := handlers.NewAuthHandler(db, googleClientID, jwtSecret)
	nh := handlers.NewNotificationHandler(db)
	uh := handlers.NewUserHandler(db, nh)
	wh := handlers.NewWorkoutHandler(db)
	ih := handlers.NewInvitationHandler(db, nh)
	ch := handlers.NewCoachHandler(db, nh)
	cph := handlers.NewCoachProfileHandler(db)
	achh := handlers.NewAchievementHandler(db, nh)
	rth := handlers.NewRatingHandler(db)
	adm := handlers.NewAdminHandler(db, nh)
	fh := handlers.NewFileHandler(db, store)

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

	// Coach request routes
	mux.HandleFunc("/api/coach-request", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			uh.GetCoachRequestStatus(w, r)
		case http.MethodPost:
			uh.RequestCoach(w, r)
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
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
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

	// Invitation routes
	mux.HandleFunc("/api/invitations", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ih.ListInvitations(w, r)
		case http.MethodPost:
			ih.CreateInvitation(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/invitations/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/respond") {
			if r.Method == http.MethodPut {
				ih.RespondInvitation(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			ih.GetInvitation(w, r)
		case http.MethodDelete:
			ih.CancelInvitation(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Notification routes
	mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			nh.ListNotifications(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/unread-count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			nh.UnreadCount(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			nh.MarkAllRead(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/action") {
			if r.Method == http.MethodPost {
				nh.ExecuteAction(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/read") {
			if r.Method == http.MethodPut {
				nh.MarkRead(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	})

	// Notification preferences routes
	mux.HandleFunc("/api/notification-preferences", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			nh.GetPreferences(w, r)
		case http.MethodPut:
			nh.UpdatePreferences(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach-student relationship routes
	mux.HandleFunc("/api/coach-students/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/end") {
			if r.Method == http.MethodPut {
				ch.EndRelationship(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	})

	// File routes
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			fh.Upload(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/files/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/download") {
			if r.Method == http.MethodGet {
				fh.Download(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method == http.MethodDelete {
			fh.Delete(w, r)
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
