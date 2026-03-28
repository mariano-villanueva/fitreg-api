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

	if _, err := db.Exec("SET time_zone = '+00:00'"); err != nil {
		return nil, fmt.Errorf("failed to set timezone: %w", err)
	}

	return db, nil
}
