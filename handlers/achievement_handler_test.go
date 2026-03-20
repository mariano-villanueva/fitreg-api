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

type mockAchievementService struct {
	listMyFn        func(coachID int64) ([]models.CoachAchievement, error)
	createFn        func(coachID int64, req models.CreateAchievementRequest) (int64, error)
	updateFn        func(achID, coachID int64, req models.UpdateAchievementRequest) error
	deleteFn        func(achID, coachID int64) error
	setVisibilityFn func(achID, coachID int64, isPublic bool) error
}

func (m *mockAchievementService) ListMy(coachID int64) ([]models.CoachAchievement, error) {
	return m.listMyFn(coachID)
}
func (m *mockAchievementService) Create(coachID int64, req models.CreateAchievementRequest) (int64, error) {
	return m.createFn(coachID, req)
}
func (m *mockAchievementService) Update(achID, coachID int64, req models.UpdateAchievementRequest) error {
	return m.updateFn(achID, coachID, req)
}
func (m *mockAchievementService) Delete(achID, coachID int64) error {
	return m.deleteFn(achID, coachID)
}
func (m *mockAchievementService) SetVisibility(achID, coachID int64, isPublic bool) error {
	return m.setVisibilityFn(achID, coachID, isPublic)
}

func newAchReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- ListMyAchievements ---

func TestAchievementHandler_ListMy_ReturnsAchievements(t *testing.T) {
	mock := &mockAchievementService{
		listMyFn: func(coachID int64) ([]models.CoachAchievement, error) {
			return []models.CoachAchievement{{ID: 1, EventName: "Marathon"}}, nil
		},
	}
	h := NewAchievementHandler(mock)
	w := httptest.NewRecorder()

	h.ListMyAchievements(w, newAchReq(http.MethodGet, "/api/coach/achievements", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.CoachAchievement
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].EventName != "Marathon" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestAchievementHandler_ListMy_ServiceError_Returns500(t *testing.T) {
	mock := &mockAchievementService{
		listMyFn: func(coachID int64) ([]models.CoachAchievement, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewAchievementHandler(mock)
	w := httptest.NewRecorder()

	h.ListMyAchievements(w, newAchReq(http.MethodGet, "/api/coach/achievements", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- CreateAchievement ---

func TestAchievementHandler_Create_ReturnsCreated(t *testing.T) {
	mock := &mockAchievementService{
		createFn: func(coachID int64, req models.CreateAchievementRequest) (int64, error) {
			return 42, nil
		},
	}
	h := NewAchievementHandler(mock)

	body, _ := json.Marshal(models.CreateAchievementRequest{
		EventName: "City Marathon",
		EventDate: "2024-03-15",
	})
	w := httptest.NewRecorder()

	h.CreateAchievement(w, newAchReq(http.MethodPost, "/api/coach/achievements", body, 10))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != float64(42) {
		t.Errorf("expected id 42, got %v", resp["id"])
	}
}

func TestAchievementHandler_Create_MissingFields_Returns400(t *testing.T) {
	h := NewAchievementHandler(&mockAchievementService{})

	body, _ := json.Marshal(models.CreateAchievementRequest{EventName: "Marathon"}) // missing event_date
	w := httptest.NewRecorder()

	h.CreateAchievement(w, newAchReq(http.MethodPost, "/api/coach/achievements", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAchievementHandler_Create_ServiceError_Returns500(t *testing.T) {
	mock := &mockAchievementService{
		createFn: func(coachID int64, req models.CreateAchievementRequest) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	h := NewAchievementHandler(mock)

	body, _ := json.Marshal(models.CreateAchievementRequest{
		EventName: "Marathon",
		EventDate: "2024-03-15",
	})
	w := httptest.NewRecorder()

	h.CreateAchievement(w, newAchReq(http.MethodPost, "/api/coach/achievements", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateAchievement ---

func TestAchievementHandler_Update_Returns200(t *testing.T) {
	mock := &mockAchievementService{
		updateFn: func(achID, coachID int64, req models.UpdateAchievementRequest) error { return nil },
	}
	h := NewAchievementHandler(mock)

	body, _ := json.Marshal(models.UpdateAchievementRequest{})
	w := httptest.NewRecorder()

	h.UpdateAchievement(w, newAchReq(http.MethodPut, "/api/coach/achievements/3", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAchievementHandler_Update_NotFound_Returns404(t *testing.T) {
	mock := &mockAchievementService{
		updateFn: func(achID, coachID int64, req models.UpdateAchievementRequest) error {
			return services.ErrNotFound
		},
	}
	h := NewAchievementHandler(mock)

	body, _ := json.Marshal(models.UpdateAchievementRequest{})
	w := httptest.NewRecorder()

	h.UpdateAchievement(w, newAchReq(http.MethodPut, "/api/coach/achievements/99", body, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- DeleteAchievement ---

func TestAchievementHandler_Delete_Returns200(t *testing.T) {
	mock := &mockAchievementService{
		deleteFn: func(achID, coachID int64) error { return nil },
	}
	h := NewAchievementHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteAchievement(w, newAchReq(http.MethodDelete, "/api/coach/achievements/3", nil, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAchievementHandler_Delete_Forbidden_Returns403(t *testing.T) {
	mock := &mockAchievementService{
		deleteFn: func(achID, coachID int64) error { return services.ErrForbidden },
	}
	h := NewAchievementHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteAchievement(w, newAchReq(http.MethodDelete, "/api/coach/achievements/3", nil, 10))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- ToggleVisibility ---

func TestAchievementHandler_ToggleVisibility_Returns200(t *testing.T) {
	mock := &mockAchievementService{
		setVisibilityFn: func(achID, coachID int64, isPublic bool) error { return nil },
	}
	h := NewAchievementHandler(mock)

	body, _ := json.Marshal(map[string]bool{"is_public": true})
	w := httptest.NewRecorder()

	h.ToggleVisibility(w, newAchReq(http.MethodPut, "/api/coach/achievements/3/visibility", body, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["is_public"] != true {
		t.Errorf("expected is_public true, got %v", resp["is_public"])
	}
}

func TestAchievementHandler_ToggleVisibility_InvalidID_Returns400(t *testing.T) {
	h := NewAchievementHandler(&mockAchievementService{})

	body, _ := json.Marshal(map[string]bool{"is_public": true})
	w := httptest.NewRecorder()

	h.ToggleVisibility(w, newAchReq(http.MethodPut, "/api/coach/achievements/abc/visibility", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
