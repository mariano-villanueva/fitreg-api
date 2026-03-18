package handlers

import (
	"bytes"
	"crypto/rand"
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

// Magic byte signatures for allowed image types.
// We read the first 12 bytes of the file to detect the real content type,
// preventing content-type spoofing attacks (e.g. an executable named .jpg).
var magicSignatures = []struct {
	mime   string
	offset int
	magic  []byte
}{
	{"image/jpeg", 0, []byte{0xFF, 0xD8, 0xFF}},
	{"image/png", 0, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
	// WebP: "RIFF" at byte 0, "WEBP" at byte 8
	{"image/webp", 0, []byte{0x52, 0x49, 0x46, 0x46}},
}

// detectMagicType reads the first 12 bytes of r and returns the matching MIME type,
// along with a new reader that includes the already-read bytes prepended.
func detectMagicType(r io.Reader) (string, io.Reader, error) {
	header := make([]byte, 12)
	n, err := io.ReadFull(r, header)
	header = header[:n]
	combined := io.MultiReader(bytes.NewReader(header), r)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", combined, err
	}
	for _, sig := range magicSignatures {
		if len(header) < sig.offset+len(sig.magic) {
			continue
		}
		if bytes.Equal(header[sig.offset:sig.offset+len(sig.magic)], sig.magic) {
			// Extra check for WebP: bytes 8-11 must be "WEBP"
			if sig.mime == "image/webp" {
				if len(header) < 12 || !bytes.Equal(header[8:12], []byte("WEBP")) {
					continue
				}
			}
			return sig.mime, combined, nil
		}
	}
	return "", combined, nil
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

	// Verify magic bytes — the Content-Type header is client-controlled and can be spoofed.
	detectedType, fileWithHeader, err := detectMagicType(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Could not read file content")
		return
	}
	if detectedType != contentType {
		writeError(w, http.StatusBadRequest, "File content does not match the declared type")
		return
	}

	uuid := generateUUID()
	storageKey := "files/" + uuid + ext

	f, err := h.svc.Upload(r.Context(), uuid, storageKey, fileWithHeader, contentType, header.Filename, header.Size, userID)
	if err != nil {
		handleServiceErr(w, err, "FileHandler.Upload", "Failed to upload file")
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

	contentType, reader, err := h.svc.Download(r.Context(), uuid, userID)
	if err != nil {
		handleServiceErr(w, err, "FileHandler.Download", "Failed to download file")
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
	if err != nil {
		handleServiceErr(w, err, "FileHandler.Delete", "Failed to delete file")
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
