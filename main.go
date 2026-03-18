package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.uber.org/fx"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
	dbprovider "github.com/fitreg/api/providers/db"
	"github.com/fitreg/api/providers/storage"
	"github.com/fitreg/api/repository"
	"github.com/fitreg/api/router"
	"github.com/fitreg/api/services"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fx.New(
		fx.Provide(
			config.Load,
			dbprovider.New,
			storage.New,
			// Workout domain
			repository.NewWorkoutRepository,
			services.NewWorkoutService,
			// File domain
			repository.NewFileRepository,
			services.NewFileService,
			// Auth + User domain (shared UserRepository)
			repository.NewUserRepository,
			services.NewAuthService,
			services.NewUserService,
			// Template domain
			repository.NewTemplateRepository,
			services.NewTemplateService,
			// CoachProfile domain
			repository.NewCoachProfileRepository,
			services.NewCoachProfileService,
			// Rating domain
			repository.NewRatingRepository,
			services.NewRatingService,
			// Notification domain
			repository.NewNotificationRepository,
			repository.NewInvitationRepository,
			services.NewNotificationService,
			services.NewInvitationService,
			// Achievement domain (Task 3)
			repository.NewAchievementRepository,
			services.NewAchievementService,
			// AssignmentMessage domain (Task 4)
			repository.NewAssignmentMessageRepository,
			services.NewAssignmentMessageService,
			// Coach domain (Task 5)
			repository.NewCoachRepository,
			services.NewCoachService,
			// Admin domain (Task 6)
			repository.NewAdminRepository,
			services.NewAdminService,
			// Weekly template domain
			repository.NewWeeklyTemplateRepository,
			services.NewWeeklyTemplateService,
			// Handlers
			handlers.NewAuthHandler,
			handlers.NewWorkoutHandler,
			handlers.NewCoachProfileHandler,
			handlers.NewRatingHandler,
			handlers.NewTemplateHandler,
			handlers.NewNotificationHandler,
			handlers.NewUserHandler,
			handlers.NewAchievementHandler,
			handlers.NewAssignmentMessageHandler,
			handlers.NewInvitationHandler,
			handlers.NewAdminHandler,
			handlers.NewCoachHandler,
			handlers.NewWeeklyTemplateHandler,
			handlers.NewFileHandler,
			router.New,
		),
		fx.Invoke(startServer),
	).Run()
}

func startServer(mux *http.ServeMux, cfg *config.Config, lc fx.Lifecycle) {
	handler := middleware.CORS(middleware.Auth(cfg.JWTSecret)(mux))
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("Server listening on :%s", cfg.ServerPort)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("Server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
