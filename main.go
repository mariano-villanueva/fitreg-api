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
