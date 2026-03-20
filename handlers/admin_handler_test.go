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

type mockAdminService struct {
	getStatsFn                func(userID int64) (map[string]int, error)
	listUsersFn               func(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error)
	updateUserFn              func(callerID, targetID int64, isCoach, isAdmin *bool) error
	listPendingAchievementsFn func(userID int64) ([]models.AdminPendingAchievement, error)
	verifyAchievementFn       func(achID, adminID int64) error
	rejectAchievementFn       func(achID, adminID int64, reason string) error
}

func (m *mockAdminService) GetStats(userID int64) (map[string]int, error) {
	return m.getStatsFn(userID)
}
func (m *mockAdminService) ListUsers(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error) {
	return m.listUsersFn(userID, search, role, sortCol, sortOrder, limit, offset)
}
func (m *mockAdminService) UpdateUser(callerID, targetID int64, isCoach, isAdmin *bool) error {
	return m.updateUserFn(callerID, targetID, isCoach, isAdmin)
}
func (m *mockAdminService) ListPendingAchievements(userID int64) ([]models.AdminPendingAchievement, error) {
	return m.listPendingAchievementsFn(userID)
}
func (m *mockAdminService) VerifyAchievement(achID, adminID int64) error {
	return m.verifyAchievementFn(achID, adminID)
}
func (m *mockAdminService) RejectAchievement(achID, adminID int64, reason string) error {
	return m.rejectAchievementFn(achID, adminID, reason)
}

func newAdminReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- GetStats ---

func TestAdminHandler_GetStats_ReturnsStats(t *testing.T) {
	mock := &mockAdminService{
		getStatsFn: func(userID int64) (map[string]int, error) {
			return map[string]int{"users": 100, "coaches": 10}, nil
		},
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.GetStats(w, newAdminReq(http.MethodGet, "/api/admin/stats", nil, 1))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["users"] != 100 {
		t.Errorf("expected users=100, got %d", resp["users"])
	}
}

func TestAdminHandler_GetStats_ServiceError_Returns500(t *testing.T) {
	mock := &mockAdminService{
		getStatsFn: func(userID int64) (map[string]int, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.GetStats(w, newAdminReq(http.MethodGet, "/api/admin/stats", nil, 1))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- ListUsers ---

func TestAdminHandler_ListUsers_ReturnsPaginated(t *testing.T) {
	mock := &mockAdminService{
		listUsersFn: func(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error) {
			return []models.AdminUser{{ID: 1, Name: "Alice"}}, 1, nil
		},
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.ListUsers(w, newAdminReq(http.MethodGet, "/api/admin/users", nil, 1))

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

func TestAdminHandler_ListUsers_ServiceError_Returns500(t *testing.T) {
	mock := &mockAdminService{
		listUsersFn: func(userID int64, search, role, sortCol, sortOrder string, limit, offset int) ([]models.AdminUser, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.ListUsers(w, newAdminReq(http.MethodGet, "/api/admin/users", nil, 1))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateUser ---

func TestAdminHandler_UpdateUser_Returns200(t *testing.T) {
	isCoach := true
	mock := &mockAdminService{
		updateUserFn: func(callerID, targetID int64, isCoach, isAdmin *bool) error { return nil },
	}
	h := NewAdminHandler(mock)

	body, _ := json.Marshal(map[string]interface{}{"is_coach": isCoach})
	w := httptest.NewRecorder()

	h.UpdateUser(w, newAdminReq(http.MethodPut, "/api/admin/users/5", body, 1))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_UpdateUser_InvalidID_Returns400(t *testing.T) {
	h := NewAdminHandler(&mockAdminService{})

	body, _ := json.Marshal(map[string]interface{}{"is_coach": true})
	w := httptest.NewRecorder()

	h.UpdateUser(w, newAdminReq(http.MethodPut, "/api/admin/users/abc", body, 1))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdminHandler_UpdateUser_Forbidden_Returns403(t *testing.T) {
	mock := &mockAdminService{
		updateUserFn: func(callerID, targetID int64, isCoach, isAdmin *bool) error {
			return services.ErrForbidden
		},
	}
	h := NewAdminHandler(mock)

	body, _ := json.Marshal(map[string]interface{}{"is_coach": true})
	w := httptest.NewRecorder()

	h.UpdateUser(w, newAdminReq(http.MethodPut, "/api/admin/users/5", body, 1))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- PendingAchievements ---

func TestAdminHandler_PendingAchievements_ReturnsList(t *testing.T) {
	mock := &mockAdminService{
		listPendingAchievementsFn: func(userID int64) ([]models.AdminPendingAchievement, error) {
			return []models.AdminPendingAchievement{{ID: 1, EventName: "Marathon"}}, nil
		},
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.PendingAchievements(w, newAdminReq(http.MethodGet, "/api/admin/achievements/pending", nil, 1))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// --- VerifyAchievement ---

func TestAdminHandler_VerifyAchievement_Returns200(t *testing.T) {
	mock := &mockAdminService{
		verifyAchievementFn: func(achID, adminID int64) error { return nil },
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.VerifyAchievement(w, newAdminReq(http.MethodPut, "/api/admin/achievements/3/verify", nil, 1))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_VerifyAchievement_NotFound_Returns404(t *testing.T) {
	mock := &mockAdminService{
		verifyAchievementFn: func(achID, adminID int64) error { return services.ErrNotFound },
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	h.VerifyAchievement(w, newAdminReq(http.MethodPut, "/api/admin/achievements/99/verify", nil, 1))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- RejectAchievement ---

func TestAdminHandler_RejectAchievement_Returns200(t *testing.T) {
	mock := &mockAdminService{
		rejectAchievementFn: func(achID, adminID int64, reason string) error { return nil },
	}
	h := NewAdminHandler(mock)

	body, _ := json.Marshal(map[string]string{"reason": "Insufficient proof"})
	w := httptest.NewRecorder()

	h.RejectAchievement(w, newAdminReq(http.MethodPut, "/api/admin/achievements/3/reject", body, 1))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandler_RejectAchievement_EmptyBody_Returns200(t *testing.T) {
	mock := &mockAdminService{
		rejectAchievementFn: func(achID, adminID int64, reason string) error { return nil },
	}
	h := NewAdminHandler(mock)
	w := httptest.NewRecorder()

	// Empty body is allowed (backwards compatibility)
	h.RejectAchievement(w, newAdminReq(http.MethodPut, "/api/admin/achievements/3/reject", nil, 1))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
