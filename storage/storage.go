package storage

import (
	"context"
	"io"
)

// Storage abstracts file storage operations.
// Implementations: S3Storage (production), LocalStorage (development).
type Storage interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
