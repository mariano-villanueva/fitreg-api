package main

import (
	"context"
	"log"
	"net/http"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/database"
	"github.com/fitreg/api/handlers"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/router"
	"github.com/fitreg/api/storage"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to MySQL database")

	// Initialize storage
	var store storage.Storage
	switch cfg.StorageProvider {
	case "s3":
		s3Store, err := storage.NewS3Storage(context.Background(), storage.S3Config{
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Endpoint:  cfg.S3Endpoint,
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		store = s3Store
		log.Println("Using S3 storage")
	default:
		localStore, err := storage.NewLocalStorage(cfg.LocalStoragePath)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
		store = localStore
		log.Printf("Using local storage at %s", cfg.LocalStoragePath)
	}

	// Construct handlers
	ah := handlers.NewAuthHandler(db, cfg)
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
	th := handlers.NewTemplateHandler(db)
	amh := handlers.NewAssignmentMessageHandler(db, nh)

	mux := router.New(wh, ch, ah, uh, ih, nh, th, achh, rth, cph, amh, adm, fh, cfg)

	// Apply middleware: CORS -> Auth
	handler := middleware.CORS(middleware.Auth(cfg.JWTSecret)(mux))

	addr := ":" + cfg.ServerPort
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
