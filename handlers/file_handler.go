package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/storage"
)

const maxFileSize = 5 << 20 // 5MB

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type FileHandler struct {
	DB      *sql.DB
	Storage storage.Storage
}

func NewFileHandler(db *sql.DB, store storage.Storage) *FileHandler {
	return &FileHandler{DB: db, Storage: store}
}

// Upload handles POST /api/files
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		writeError(w, http.StatusBadRequest, "File too large (max 5MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedContentTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid file type. Allowed: JPG, PNG, WebP")
		return
	}

	if header.Size > maxFileSize {
		writeError(w, http.StatusBadRequest, "File too large (max 5MB)")
		return
	}

	uuid := generateUUID()
	storageKey := fmt.Sprintf("files/%s%s", uuid, ext)

	// Upload to storage
	if err := h.Storage.Upload(r.Context(), storageKey, file, contentType); err != nil {
		log.Printf("ERROR uploading file to storage: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	// Insert DB record
	result, err := h.DB.Exec(
		"INSERT INTO files (uuid, user_id, original_name, content_type, size_bytes, storage_key) VALUES (?, ?, ?, ?, ?, ?)",
		uuid, userID, header.Filename, contentType, header.Size, storageKey,
	)
	if err != nil {
		log.Printf("ERROR inserting file record: %v", err)
		// Best-effort rollback: delete from storage
		if delErr := h.Storage.Delete(r.Context(), storageKey); delErr != nil {
			log.Printf("ERROR rolling back storage upload: %v", delErr)
		}
		writeError(w, http.StatusInternalServerError, "Failed to save file record")
		return
	}

	id, _ := result.LastInsertId()
	f := models.File{
		ID:           id,
		UUID:         uuid,
		OriginalName: header.Filename,
		ContentType:  contentType,
		SizeBytes:    header.Size,
		URL:          fmt.Sprintf("/api/files/%s/download", uuid),
		CreatedAt:    time.Now(),
	}

	writeJSON(w, http.StatusCreated, f)
}

// Download handles GET /api/files/{uuid}/download
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract UUID from path: /api/files/{uuid}/download
	path := strings.TrimPrefix(r.URL.Path, "/api/files/")
	uuid := strings.TrimSuffix(path, "/download")
	if uuid == "" || uuid == path {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	var f models.File
	err := h.DB.QueryRow(
		"SELECT id, uuid, content_type, storage_key, original_name FROM files WHERE uuid = ?",
		uuid,
	).Scan(&f.ID, &f.UUID, &f.ContentType, &f.StorageKey, &f.OriginalName)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		log.Printf("ERROR fetching file record: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch file")
		return
	}

	reader, err := h.Storage.Download(r.Context(), f.StorageKey)
	if err != nil {
		log.Printf("ERROR downloading file from storage: %v", err)
		writeError(w, http.StatusNotFound, "File not found in storage")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	io.Copy(w, reader)
}

// Delete handles DELETE /api/files/{uuid}
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract UUID from path: /api/files/{uuid}
	uuid := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	var fileUserID int64
	var storageKey string
	err := h.DB.QueryRow(
		"SELECT user_id, storage_key FROM files WHERE uuid = ?",
		uuid,
	).Scan(&fileUserID, &storageKey)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		log.Printf("ERROR fetching file for delete: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch file")
		return
	}

	if fileUserID != userID {
		writeError(w, http.StatusForbidden, "Not authorized to delete this file")
		return
	}

	// Delete from storage first
	if err := h.Storage.Delete(r.Context(), storageKey); err != nil {
		log.Printf("ERROR deleting file from storage: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete file from storage")
		return
	}

	// Then delete DB record
	if _, err := h.DB.Exec("DELETE FROM files WHERE uuid = ?", uuid); err != nil {
		log.Printf("ERROR deleting file record: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete file record")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := fmt.Sprintf("%x", b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
