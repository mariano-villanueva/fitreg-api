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
			// Repositories
			repository.NewWorkoutRepository,
			repository.NewFileRepository,
			repository.NewUserRepository,
			repository.NewTemplateRepository,
			repository.NewCoachProfileRepository,
			repository.NewRatingRepository,
			repository.NewNotificationRepository,
			repository.NewInvitationRepository,
			repository.NewAchievementRepository,
			repository.NewAssignmentMessageRepository,
			repository.NewCoachRepository,
			repository.NewAdminRepository,
			repository.NewWeeklyTemplateRepository,
			// Services — annotated so fx resolves interface deps in handlers
			fx.Annotate(services.NewWorkoutService,
				fx.As(new(handlers.WorkoutServicer))),
			fx.Annotate(services.NewFileService,
				fx.As(new(handlers.FileServicer))),
			fx.Annotate(services.NewAuthService,
				fx.As(new(handlers.AuthServicer))),
			fx.Annotate(services.NewUserService,
				fx.As(new(handlers.UserServicer))),
			fx.Annotate(services.NewTemplateService,
				fx.As(new(handlers.TemplateServicer))),
			fx.Annotate(services.NewCoachProfileService,
				fx.As(new(handlers.CoachProfileServicer))),
			fx.Annotate(services.NewRatingService,
				fx.As(new(handlers.RatingServicer))),
			// NotificationService: keep concrete type (used by other services)
			// and also expose as handler interfaces via wrappers
			services.NewNotificationService,
			func(s *services.NotificationService) handlers.NotificationServicer { return s },
			func(s *services.NotificationService) handlers.NotificationCreator { return s },
			fx.Annotate(services.NewInvitationService,
				fx.As(new(handlers.InvitationServicer))),
			fx.Annotate(services.NewAchievementService,
				fx.As(new(handlers.AchievementServicer))),
			fx.Annotate(services.NewAssignmentMessageService,
				fx.As(new(handlers.AssignmentMessageServicer))),
			fx.Annotate(services.NewCoachService,
				fx.As(new(handlers.CoachServicer))),
			fx.Annotate(services.NewAdminService,
				fx.As(new(handlers.AdminServicer))),
			fx.Annotate(services.NewWeeklyTemplateService,
				fx.As(new(handlers.WeeklyTemplateServicer))),
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
