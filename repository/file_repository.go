package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fitreg/api/models"
)

type fileRepository struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(uuid string, userID int64, name, contentType string, size int64, storageKey string) (models.File, error) {
	result, err := r.db.Exec(
		"INSERT INTO files (uuid, user_id, original_name, content_type, size_bytes, storage_key) VALUES (?, ?, ?, ?, ?, ?)",
		uuid, userID, name, contentType, size, storageKey,
	)
	if err != nil {
		return models.File{}, err
	}
	id, _ := result.LastInsertId()
	return models.File{
		ID:           id,
		UUID:         uuid,
		OriginalName: name,
		ContentType:  contentType,
		SizeBytes:    size,
		URL:          fmt.Sprintf("/api/files/%s/download", uuid),
		CreatedAt:    time.Now(),
	}, nil
}

func (r *fileRepository) GetByUUID(uuid string) (models.File, error) {
	var f models.File
	err := r.db.QueryRow(
		"SELECT id, uuid, content_type, storage_key, original_name FROM files WHERE uuid = ?",
		uuid,
	).Scan(&f.ID, &f.UUID, &f.ContentType, &f.StorageKey, &f.OriginalName)
	return f, err
}

func (r *fileRepository) GetOwnerAndKey(uuid string) (int64, string, error) {
	var userID int64
	var storageKey string
	err := r.db.QueryRow(
		"SELECT user_id, storage_key FROM files WHERE uuid = ?",
		uuid,
	).Scan(&userID, &storageKey)
	return userID, storageKey, err
}

func (r *fileRepository) Delete(uuid string) error {
	_, err := r.db.Exec("DELETE FROM files WHERE uuid = ?", uuid)
	return err
}
