package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type mockRatingService struct {
	upsertFn func(coachID, studentID int64, req models.UpsertRatingRequest) error
	listFn   func(coachID int64) ([]models.CoachRating, error)
}

func (m *mockRatingService) Upsert(coachID, studentID int64, req models.UpsertRatingRequest) error {
	return m.upsertFn(coachID, studentID, req)
}
func (m *mockRatingService) List(coachID int64) ([]models.CoachRating, error) {
	return m.listFn(coachID)
}

func newRatingReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- UpsertRating ---

func TestRatingHandler_Upsert_Returns200(t *testing.T) {
	mock := &mockRatingService{
		upsertFn: func(coachID, studentID int64, req models.UpsertRatingRequest) error { return nil },
	}
	h := NewRatingHandler(mock)

	body, _ := json.Marshal(models.UpsertRatingRequest{Rating: 8})
	w := httptest.NewRecorder()

	h.UpsertRating(w, newRatingReq(http.MethodPost, "/api/coaches/5/ratings", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRatingHandler_Upsert_InvalidCoachID_Returns400(t *testing.T) {
	h := NewRatingHandler(&mockRatingService{})

	body, _ := json.Marshal(models.UpsertRatingRequest{Rating: 8})
	w := httptest.NewRecorder()

	h.UpsertRating(w, newRatingReq(http.MethodPost, "/api/coaches/abc/ratings", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRatingHandler_Upsert_NotStudent_Returns403(t *testing.T) {
	mock := &mockRatingService{
		upsertFn: func(coachID, studentID int64, req models.UpsertRatingRequest) error {
			return services.ErrNotStudent
		},
	}
	h := NewRatingHandler(mock)

	body, _ := json.Marshal(models.UpsertRatingRequest{Rating: 8})
	w := httptest.NewRecorder()

	h.UpsertRating(w, newRatingReq(http.MethodPost, "/api/coaches/5/ratings", body, 42))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRatingHandler_Upsert_InvalidRating_Returns400(t *testing.T) {
	mock := &mockRatingService{
		upsertFn: func(coachID, studentID int64, req models.UpsertRatingRequest) error {
			return services.ErrInvalidRating
		},
	}
	h := NewRatingHandler(mock)

	body, _ := json.Marshal(models.UpsertRatingRequest{Rating: 99})
	w := httptest.NewRecorder()

	h.UpsertRating(w, newRatingReq(http.MethodPost, "/api/coaches/5/ratings", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRatingHandler_Upsert_ServiceError_Returns500(t *testing.T) {
	mock := &mockRatingService{
		upsertFn: func(coachID, studentID int64, req models.UpsertRatingRequest) error {
			return errors.New("db error")
		},
	}
	h := NewRatingHandler(mock)

	body, _ := json.Marshal(models.UpsertRatingRequest{Rating: 8})
	w := httptest.NewRecorder()

	h.UpsertRating(w, newRatingReq(http.MethodPost, "/api/coaches/5/ratings", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetRatings ---

func TestRatingHandler_GetRatings_ReturnsRatings(t *testing.T) {
	mock := &mockRatingService{
		listFn: func(coachID int64) ([]models.CoachRating, error) {
			return []models.CoachRating{{CoachID: coachID, Rating: 9}}, nil
		},
	}
	h := NewRatingHandler(mock)
	w := httptest.NewRecorder()

	// GetRatings doesn't require auth (no userID check), use zero userID
	r := httptest.NewRequest(http.MethodGet, "/api/coaches/5/ratings", nil)
	h.GetRatings(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.CoachRating
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Rating != 9 {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestRatingHandler_GetRatings_InvalidCoachID_Returns400(t *testing.T) {
	h := NewRatingHandler(&mockRatingService{})
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/coaches/abc/ratings", nil)
	h.GetRatings(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRatingHandler_GetRatings_ServiceError_Returns500(t *testing.T) {
	mock := &mockRatingService{
		listFn: func(coachID int64) ([]models.CoachRating, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewRatingHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/coaches/5/ratings", nil)
	h.GetRatings(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
