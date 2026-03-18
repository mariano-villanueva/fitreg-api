package router

import (
	"net/http"
	"strings"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
)

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
	weeklyTemplate *handlers.WeeklyTemplateHandler,
	file *handlers.FileHandler,
	cfg *config.Config,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Auth routes (public, rate-limited: 10 req/IP/min)
	mux.Handle("/api/auth/google", middleware.RateLimitAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			auth.GoogleLogin(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))

	// User profile routes
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			user.GetProfile(w, r)
		case http.MethodPut:
			user.UpdateProfile(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Avatar routes
	mux.HandleFunc("/api/me/avatar", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			user.UploadAvatar(w, r)
		case http.MethodDelete:
			user.DeleteAvatar(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Workout routes
	mux.HandleFunc("/api/workouts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workout.ListWorkouts(w, r)
		case http.MethodPost:
			workout.CreateWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/workouts/", func(w http.ResponseWriter, r *http.Request) {
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

	// Coach request routes
	mux.HandleFunc("/api/coach-request", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			user.GetCoachRequestStatus(w, r)
		case http.MethodPost:
			user.RequestCoach(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach student routes
	mux.HandleFunc("/api/coach/students", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			coach.ListStudents(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/students/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/workouts") {
			if r.Method == http.MethodGet {
				coach.GetStudentWorkouts(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	})

	// Coach daily summary route
	mux.HandleFunc("/api/coach/daily-summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			coach.GetDailySummary(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach template routes
	mux.HandleFunc("/api/coach/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			template.List(w, r)
		case http.MethodPost:
			template.Create(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/templates/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			template.Get(w, r)
		case http.MethodPut:
			template.Update(w, r)
		case http.MethodDelete:
			template.Delete(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Weekly template routes
	mux.HandleFunc("/api/coach/weekly-templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			weeklyTemplate.List(w, r)
		case http.MethodPost:
			weeklyTemplate.Create(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/weekly-templates/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/days") {
			if r.Method == http.MethodPut {
				weeklyTemplate.PutDays(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/assign") {
			if r.Method == http.MethodPost {
				weeklyTemplate.Assign(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			weeklyTemplate.Get(w, r)
		case http.MethodPut:
			weeklyTemplate.UpdateMeta(w, r)
		case http.MethodDelete:
			weeklyTemplate.Delete(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach assigned workouts routes
	mux.HandleFunc("/api/coach/assigned-workouts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			coach.ListAssignedWorkouts(w, r)
		case http.MethodPost:
			coach.CreateAssignedWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/assigned-workouts/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			coach.GetAssignedWorkout(w, r)
		case http.MethodPut:
			coach.UpdateAssignedWorkout(w, r)
		case http.MethodDelete:
			coach.DeleteAssignedWorkout(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Student assigned workouts routes
	mux.HandleFunc("/api/my-assigned-workouts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			coach.GetMyAssignedWorkouts(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/my-assigned-workouts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/status") {
			coach.UpdateAssignedWorkoutStatus(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach profile routes
	mux.HandleFunc("/api/coach/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			coachProfile.UpdateCoachProfile(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach achievements routes
	mux.HandleFunc("/api/coach/achievements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			achievement.ListMyAchievements(w, r)
		case http.MethodPost:
			achievement.CreateAchievement(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coach/achievements/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/visibility") {
			achievement.ToggleVisibility(w, r)
			return
		}
		switch r.Method {
		case http.MethodPut:
			achievement.UpdateAchievement(w, r)
		case http.MethodDelete:
			achievement.DeleteAchievement(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach directory routes
	mux.HandleFunc("/api/coaches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			coachProfile.ListCoaches(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/coaches/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ratings") {
			switch r.Method {
			case http.MethodGet:
				rating.GetRatings(w, r)
			case http.MethodPost:
				rating.UpsertRating(w, r)
			default:
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method == http.MethodGet {
			coachProfile.GetCoachProfile(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Admin routes
	mux.HandleFunc("/api/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.GetStats(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.ListUsers(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			admin.UpdateUser(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/achievements/pending", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.PendingAchievements(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/admin/achievements/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			if strings.HasSuffix(r.URL.Path, "/verify") {
				admin.VerifyAchievement(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/reject") {
				admin.RejectAchievement(w, r)
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
			invitation.ListInvitations(w, r)
		case http.MethodPost:
			invitation.CreateInvitation(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/invitations/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/respond") {
			if r.Method == http.MethodPut {
				invitation.RespondInvitation(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			invitation.GetInvitation(w, r)
		case http.MethodDelete:
			invitation.CancelInvitation(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Notification routes
	mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			notification.ListNotifications(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/unread-count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			notification.UnreadCount(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			notification.MarkAllRead(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/action") {
			if r.Method == http.MethodPost {
				notification.ExecuteAction(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/read") {
			if r.Method == http.MethodPut {
				notification.MarkRead(w, r)
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
			notification.GetPreferences(w, r)
		case http.MethodPut:
			notification.UpdatePreferences(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Coach-student relationship routes
	mux.HandleFunc("/api/coach-students/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/end") {
			if r.Method == http.MethodPut {
				coach.EndRelationship(w, r)
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
			file.Upload(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/files/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/download") {
			if r.Method == http.MethodGet {
				file.Download(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method == http.MethodDelete {
			file.Delete(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Assignment messages
	mux.HandleFunc("/api/assignment-messages/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/read") {
			if r.Method == http.MethodPut {
				assignmentMessage.MarkRead(w, r)
			} else {
				http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			assignmentMessage.ListMessages(w, r)
		case http.MethodPost:
			assignmentMessage.SendMessage(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Assignment detail (both coach and student)
	mux.HandleFunc("/api/assigned-workout-detail/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			assignmentMessage.GetAssignedWorkoutDetail(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	// Health check (public)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux
}
