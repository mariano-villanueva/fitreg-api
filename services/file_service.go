package services

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/providers/storage"
	"github.com/fitreg/api/repository"
)

type FileService struct {
	repo  repository.FileRepository
	store storage.Storage
}

func NewFileService(repo repository.FileRepository, store storage.Storage) *FileService {
	return &FileService{repo: repo, store: store}
}

func (s *FileService) Upload(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error) {
	if err := s.store.Upload(ctx, storageKey, file, contentType); err != nil {
		log.Printf("ERROR uploading file to storage: %v", err)
		return models.File{}, err
	}
	f, err := s.repo.Create(uuid, userID, originalName, contentType, size, storageKey)
	if err != nil {
		log.Printf("ERROR inserting file record: %v", err)
		// Best-effort rollback
		if delErr := s.store.Delete(ctx, storageKey); delErr != nil {
			log.Printf("ERROR rolling back storage upload: %v", delErr)
		}
		return models.File{}, err
	}
	return f, nil
}

func (s *FileService) Download(ctx context.Context, uuid string) (string, io.ReadCloser, error) {
	f, err := s.repo.GetByUUID(uuid)
	if err != nil {
		return "", nil, err
	}
	reader, err := s.store.Download(ctx, f.StorageKey)
	if err != nil {
		log.Printf("ERROR downloading file from storage: %v", err)
		return "", nil, err
	}
	return f.ContentType, reader, nil
}

var ErrForbidden = errors.New("forbidden")

func (s *FileService) Delete(ctx context.Context, uuid string, userID int64) error {
	ownerID, storageKey, err := s.repo.GetOwnerAndKey(uuid)
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrForbidden
	}
	if err := s.store.Delete(ctx, storageKey); err != nil {
		log.Printf("ERROR deleting file from storage: %v", err)
		return err
	}
	return s.repo.Delete(uuid)
}
