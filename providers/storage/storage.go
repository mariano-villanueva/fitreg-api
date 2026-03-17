package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/fitreg/api/config"
)

// Storage abstracts file storage operations.
// Implementations: S3Storage (production), LocalStorage (development).
type Storage interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// New constructs the appropriate Storage implementation based on cfg.StorageProvider.
// FX will inject *config.Config automatically.
func New(cfg *config.Config) (Storage, error) {
	switch cfg.StorageProvider {
	case "s3":
		s3Store, err := NewS3Storage(context.Background(), S3Config{
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Endpoint:  cfg.S3Endpoint,
		})
		if err != nil {
			return nil, fmt.Errorf("init s3 storage: %w", err)
		}
		return s3Store, nil
	default:
		localStore, err := NewLocalStorage(cfg.LocalStoragePath)
		if err != nil {
			return nil, fmt.Errorf("init local storage: %w", err)
		}
		return localStore, nil
	}
}
