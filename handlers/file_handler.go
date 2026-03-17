package handlers

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/services"
)

const maxFileSize = 5 << 20 // 5MB

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type FileHandler struct {
	svc *services.FileService
}

func NewFileHandler(svc *services.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

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
	storageKey := "files/" + uuid + ext

	f, err := h.svc.Upload(r.Context(), uuid, storageKey, file, contentType, header.Filename, header.Size, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/files/")
	uuid := strings.TrimSuffix(path, "/download")
	if uuid == "" || uuid == path {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	contentType, reader, err := h.svc.Download(r.Context(), uuid)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "File not found in storage")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("ERROR streaming file %s: %v", uuid, err)
	}
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	uuid := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if uuid == "" {
		writeError(w, http.StatusBadRequest, "Invalid file UUID")
		return
	}

	err := h.svc.Delete(r.Context(), uuid, userID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		writeError(w, http.StatusForbidden, "Not authorized to delete this file")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete file")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

// generateUUID generates a random UUID v4 string.
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("ERROR generating UUID: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := fmt.Sprintf("%x", b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
