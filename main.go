package main

import (
	"log"
	"net/http"

	"github.com/fitreg/api/config"
	"github.com/fitreg/api/database"
	"github.com/fitreg/api/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to MySQL database")

	handler := router.New(db, cfg.GoogleClientID, cfg.JWTSecret)

	addr := ":" + cfg.ServerPort
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
