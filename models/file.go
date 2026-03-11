package models

import "time"

type File struct {
	ID           int64     `json:"id"`
	UUID         string    `json:"uuid"`
	UserID       int64     `json:"-"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	StorageKey   string    `json:"-"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
}
