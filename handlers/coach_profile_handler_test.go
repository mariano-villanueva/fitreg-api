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

type mockCoachProfileService struct {
	updateProfileFn  func(coachID int64, req models.UpdateCoachProfileRequest) error
	listCoachesFn    func(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error)
	getCoachProfileFn func(coachID, requestingUserID int64) (models.CoachPublicProfile, error)
}

func (m *mockCoachProfileService) UpdateProfile(coachID int64, req models.UpdateCoachProfileRequest) error {
	return m.updateProfileFn(coachID, req)
}
func (m *mockCoachProfileService) ListCoaches(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
	return m.listCoachesFn(search, locality, level, sortBy, limit, offset)
}
func (m *mockCoachProfileService) GetCoachProfile(coachID, requestingUserID int64) (models.CoachPublicProfile, error) {
	return m.getCoachProfileFn(coachID, requestingUserID)
}

func newCPReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if userID != 0 {
		return r.WithContext(middleware.WithUserID(r.Context(), userID))
	}
	return r
}

// --- UpdateCoachProfile ---

func TestCoachProfileHandler_UpdateProfile_Returns200(t *testing.T) {
	mock := &mockCoachProfileService{
		updateProfileFn: func(coachID int64, req models.UpdateCoachProfileRequest) error { return nil },
	}
	h := NewCoachProfileHandler(mock)

	body, _ := json.Marshal(models.UpdateCoachProfileRequest{CoachDescription: "10 years experience"})
	w := httptest.NewRecorder()

	h.UpdateCoachProfile(w, newCPReq(http.MethodPut, "/api/coach/profile", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCoachProfileHandler_UpdateProfile_InvalidBody_Returns400(t *testing.T) {
	h := NewCoachProfileHandler(&mockCoachProfileService{})
	w := httptest.NewRecorder()

	r := newCPReq(http.MethodPut, "/api/coach/profile", []byte("{bad json"), 10)
	h.UpdateCoachProfile(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachProfileHandler_UpdateProfile_NotCoach_Returns403(t *testing.T) {
	mock := &mockCoachProfileService{
		updateProfileFn: func(coachID int64, req models.UpdateCoachProfileRequest) error {
			return services.ErrNotCoach
		},
	}
	h := NewCoachProfileHandler(mock)

	body, _ := json.Marshal(models.UpdateCoachProfileRequest{})
	w := httptest.NewRecorder()

	h.UpdateCoachProfile(w, newCPReq(http.MethodPut, "/api/coach/profile", body, 10))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCoachProfileHandler_UpdateProfile_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachProfileService{
		updateProfileFn: func(coachID int64, req models.UpdateCoachProfileRequest) error {
			return errors.New("db error")
		},
	}
	h := NewCoachProfileHandler(mock)

	body, _ := json.Marshal(models.UpdateCoachProfileRequest{})
	w := httptest.NewRecorder()

	h.UpdateCoachProfile(w, newCPReq(http.MethodPut, "/api/coach/profile", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- ListCoaches ---

func TestCoachProfileHandler_ListCoaches_ReturnsPaginated(t *testing.T) {
	mock := &mockCoachProfileService{
		listCoachesFn: func(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
			return []models.CoachListItem{{ID: 1, Name: "Alice"}}, 1, nil
		},
	}
	h := NewCoachProfileHandler(mock)
	w := httptest.NewRecorder()

	// ListCoaches has no auth requirement
	r := httptest.NewRequest(http.MethodGet, "/api/coaches", nil)
	h.ListCoaches(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["total"] != float64(1) {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
}

func TestCoachProfileHandler_ListCoaches_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachProfileService{
		listCoachesFn: func(search, locality, level, sortBy string, limit, offset int) ([]models.CoachListItem, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	h := NewCoachProfileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/coaches", nil)
	h.ListCoaches(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetCoachProfile ---

func TestCoachProfileHandler_GetCoachProfile_ReturnsProfile(t *testing.T) {
	mock := &mockCoachProfileService{
		getCoachProfileFn: func(coachID, requestingUserID int64) (models.CoachPublicProfile, error) {
			return models.CoachPublicProfile{ID: coachID, Name: "Alice"}, nil
		},
	}
	h := NewCoachProfileHandler(mock)
	w := httptest.NewRecorder()

	h.GetCoachProfile(w, newCPReq(http.MethodGet, "/api/coaches/5", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.CoachPublicProfile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 5 || resp.Name != "Alice" {
		t.Errorf("unexpected profile: %+v", resp)
	}
}

func TestCoachProfileHandler_GetCoachProfile_InvalidID_Returns400(t *testing.T) {
	h := NewCoachProfileHandler(&mockCoachProfileService{})
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/coaches/abc", nil)
	h.GetCoachProfile(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachProfileHandler_GetCoachProfile_NotFound_Returns404(t *testing.T) {
	mock := &mockCoachProfileService{
		getCoachProfileFn: func(coachID, requestingUserID int64) (models.CoachPublicProfile, error) {
			return models.CoachPublicProfile{}, services.ErrNotFound
		},
	}
	h := NewCoachProfileHandler(mock)
	w := httptest.NewRecorder()

	r := httptest.NewRequest(http.MethodGet, "/api/coaches/99", nil)
	h.GetCoachProfile(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
