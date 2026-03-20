package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type mockFileService struct {
	uploadFn   func(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error)
	downloadFn func(ctx context.Context, uuid string, userID int64) (string, io.ReadCloser, error)
	deleteFn   func(ctx context.Context, uuid string, userID int64) error
}

func (m *mockFileService) Upload(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error) {
	return m.uploadFn(ctx, uuid, storageKey, file, contentType, originalName, size, userID)
}
func (m *mockFileService) Download(ctx context.Context, uuid string, userID int64) (string, io.ReadCloser, error) {
	return m.downloadFn(ctx, uuid, userID)
}
func (m *mockFileService) Delete(ctx context.Context, uuid string, userID int64) error {
	return m.deleteFn(ctx, uuid, userID)
}

// minimalPNG is the smallest valid PNG (1x1 transparent pixel).
var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

// buildMultipartRequest creates a multipart/form-data request with a file field.
func buildMultipartRequest(fieldName, fileName, contentType string, content []byte, userID int64) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, _ := w.CreateFormFile(fieldName, fileName)
	part.Write(content)
	w.Close()

	r := httptest.NewRequest(http.MethodPost, "/api/files", body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	// Manually set the content type on the file part by patching the header
	// (multipart.CreateFormFile sets application/octet-stream; we need image/png)
	// Instead, rebuild with the correct part header:
	body2 := &bytes.Buffer{}
	mw := multipart.NewWriter(body2)
	h2 := make(map[string][]string)
	h2["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + fileName + `"`}
	h2["Content-Type"] = []string{contentType}
	pw, _ := mw.CreatePart(h2)
	pw.Write(content)
	mw.Close()

	r2 := httptest.NewRequest(http.MethodPost, "/api/files", body2)
	r2.Header.Set("Content-Type", mw.FormDataContentType())
	if userID != 0 {
		return r2.WithContext(middleware.WithUserID(r2.Context(), userID))
	}
	return r2
}

// --- Upload ---

func TestFileHandler_Upload_ReturnsCreated(t *testing.T) {
	mock := &mockFileService{
		uploadFn: func(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error) {
			return models.File{UUID: uuid, ContentType: contentType}, nil
		},
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := buildMultipartRequest("file", "test.png", "image/png", minimalPNG, 42)
	h.Upload(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileHandler_Upload_MissingFileField_Returns400(t *testing.T) {
	h := NewFileHandler(&mockFileService{})

	// Form with no file field
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	mw.WriteField("other", "value")
	mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/api/files", body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))

	w := httptest.NewRecorder()
	h.Upload(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFileHandler_Upload_InvalidContentType_Returns400(t *testing.T) {
	h := NewFileHandler(&mockFileService{})
	w := httptest.NewRecorder()

	// text/plain is not allowed
	r := buildMultipartRequest("file", "test.txt", "text/plain", []byte("hello"), 42)
	h.Upload(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFileHandler_Upload_MagicBytesMismatch_Returns400(t *testing.T) {
	h := NewFileHandler(&mockFileService{})
	w := httptest.NewRecorder()

	// Claims to be PNG but content is not a valid PNG
	r := buildMultipartRequest("file", "fake.png", "image/png", []byte("this is not a png"), 42)
	h.Upload(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFileHandler_Upload_ServiceError_Returns500(t *testing.T) {
	mock := &mockFileService{
		uploadFn: func(ctx context.Context, uuid, storageKey string, file io.Reader, contentType, originalName string, size int64, userID int64) (models.File, error) {
			return models.File{}, errors.New("storage error")
		},
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := buildMultipartRequest("file", "test.png", "image/png", minimalPNG, 42)
	h.Upload(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Download ---

func TestFileHandler_Download_StreamsContent(t *testing.T) {
	mock := &mockFileService{
		downloadFn: func(ctx context.Context, uuid string, userID int64) (string, io.ReadCloser, error) {
			return "image/png", io.NopCloser(strings.NewReader("file-content")), nil
		},
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/files/abc-uuid/download", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Download(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %q", ct)
	}
	if w.Body.String() != "file-content" {
		t.Errorf("expected body 'file-content', got %q", w.Body.String())
	}
}

func TestFileHandler_Download_MissingUUID_Returns400(t *testing.T) {
	h := NewFileHandler(&mockFileService{})
	w := httptest.NewRecorder()

	// Path that strips to empty UUID
	r := httptest.NewRequest(http.MethodGet, "/api/files//download", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Download(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFileHandler_Download_NotFound_Returns404(t *testing.T) {
	mock := &mockFileService{
		downloadFn: func(ctx context.Context, uuid string, userID int64) (string, io.ReadCloser, error) {
			return "", nil, services.ErrNotFound
		},
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/files/missing-uuid/download", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Download(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Delete ---

func TestFileHandler_Delete_Returns200(t *testing.T) {
	mock := &mockFileService{
		deleteFn: func(ctx context.Context, uuid string, userID int64) error { return nil },
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodDelete, "/api/files/abc-uuid", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestFileHandler_Delete_Forbidden_Returns403(t *testing.T) {
	mock := &mockFileService{
		deleteFn: func(ctx context.Context, uuid string, userID int64) error { return services.ErrForbidden },
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodDelete, "/api/files/abc-uuid", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Delete(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestFileHandler_Delete_ServiceError_Returns500(t *testing.T) {
	mock := &mockFileService{
		deleteFn: func(ctx context.Context, uuid string, userID int64) error { return errors.New("storage error") },
	}
	h := NewFileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodDelete, "/api/files/abc-uuid", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), 42))
	h.Delete(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
