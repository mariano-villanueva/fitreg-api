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

type mockNotificationService struct {
	listFn              func(userID int64, limit, offset int) ([]models.Notification, error)
	unreadCountFn       func(userID int64) (int, error)
	markReadFn          func(notifID, userID int64) (bool, error)
	markAllReadFn       func(userID int64) error
	executeActionFn     func(notifID, userID int64, action string) error
	getPreferencesFn    func(userID int64) (models.NotificationPreferences, error)
	updatePreferencesFn func(userID int64, req models.UpdateNotificationPreferencesRequest) error
}

func (m *mockNotificationService) List(userID int64, limit, offset int) ([]models.Notification, error) {
	return m.listFn(userID, limit, offset)
}
func (m *mockNotificationService) UnreadCount(userID int64) (int, error) {
	return m.unreadCountFn(userID)
}
func (m *mockNotificationService) MarkRead(notifID, userID int64) (bool, error) {
	return m.markReadFn(notifID, userID)
}
func (m *mockNotificationService) MarkAllRead(userID int64) error {
	return m.markAllReadFn(userID)
}
func (m *mockNotificationService) ExecuteAction(notifID, userID int64, action string) error {
	return m.executeActionFn(notifID, userID, action)
}
func (m *mockNotificationService) GetPreferences(userID int64) (models.NotificationPreferences, error) {
	return m.getPreferencesFn(userID)
}
func (m *mockNotificationService) UpdatePreferences(userID int64, req models.UpdateNotificationPreferencesRequest) error {
	return m.updatePreferencesFn(userID, req)
}

func newNotifReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- ListNotifications ---

func TestNotificationHandler_List_ReturnsNotifications(t *testing.T) {
	mock := &mockNotificationService{
		listFn: func(userID int64, limit, offset int) ([]models.Notification, error) {
			return []models.Notification{{ID: 1}, {ID: 2}}, nil
		},
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.ListNotifications(w, newNotifReq(http.MethodGet, "/api/notifications", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.Notification
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(resp))
	}
}

func TestNotificationHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockNotificationService{
		listFn: func(userID int64, limit, offset int) ([]models.Notification, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.ListNotifications(w, newNotifReq(http.MethodGet, "/api/notifications", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UnreadCount ---

func TestNotificationHandler_UnreadCount_ReturnsCount(t *testing.T) {
	mock := &mockNotificationService{
		unreadCountFn: func(userID int64) (int, error) { return 5, nil },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.UnreadCount(w, newNotifReq(http.MethodGet, "/api/notifications/unread-count", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["count"] != 5 {
		t.Errorf("expected count 5, got %d", resp["count"])
	}
}

func TestNotificationHandler_UnreadCount_ServiceError_Returns500(t *testing.T) {
	mock := &mockNotificationService{
		unreadCountFn: func(userID int64) (int, error) { return 0, errors.New("db error") },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.UnreadCount(w, newNotifReq(http.MethodGet, "/api/notifications/unread-count", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- MarkRead ---

func TestNotificationHandler_MarkRead_Returns200(t *testing.T) {
	mock := &mockNotificationService{
		markReadFn: func(notifID, userID int64) (bool, error) { return true, nil },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.MarkRead(w, newNotifReq(http.MethodPut, "/api/notifications/3/read", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkRead_NotFound_Returns404(t *testing.T) {
	mock := &mockNotificationService{
		markReadFn: func(notifID, userID int64) (bool, error) { return false, nil },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.MarkRead(w, newNotifReq(http.MethodPut, "/api/notifications/3/read", nil, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkRead_InvalidID_Returns400(t *testing.T) {
	h := NewNotificationHandler(&mockNotificationService{})
	w := httptest.NewRecorder()

	h.MarkRead(w, newNotifReq(http.MethodPut, "/api/notifications/abc/read", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- MarkAllRead ---

func TestNotificationHandler_MarkAllRead_Returns200(t *testing.T) {
	mock := &mockNotificationService{
		markAllReadFn: func(userID int64) error { return nil },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.MarkAllRead(w, newNotifReq(http.MethodPut, "/api/notifications/read-all", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNotificationHandler_MarkAllRead_ServiceError_Returns500(t *testing.T) {
	mock := &mockNotificationService{
		markAllReadFn: func(userID int64) error { return errors.New("db error") },
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.MarkAllRead(w, newNotifReq(http.MethodPut, "/api/notifications/read-all", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- ExecuteAction ---

func TestNotificationHandler_ExecuteAction_Returns200(t *testing.T) {
	mock := &mockNotificationService{
		executeActionFn: func(notifID, userID int64, action string) error { return nil },
	}
	h := NewNotificationHandler(mock)

	body, _ := json.Marshal(models.NotificationActionRequest{Action: "accept"})
	w := httptest.NewRecorder()

	h.ExecuteAction(w, newNotifReq(http.MethodPost, "/api/notifications/5/action", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNotificationHandler_ExecuteAction_NotFound_Returns404(t *testing.T) {
	mock := &mockNotificationService{
		executeActionFn: func(notifID, userID int64, action string) error { return services.ErrNotFound },
	}
	h := NewNotificationHandler(mock)

	body, _ := json.Marshal(models.NotificationActionRequest{Action: "accept"})
	w := httptest.NewRecorder()

	h.ExecuteAction(w, newNotifReq(http.MethodPost, "/api/notifications/5/action", body, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- GetPreferences ---

func TestNotificationHandler_GetPreferences_ReturnsPrefs(t *testing.T) {
	mock := &mockNotificationService{
		getPreferencesFn: func(userID int64) (models.NotificationPreferences, error) {
			return models.NotificationPreferences{UserID: userID}, nil
		},
	}
	h := NewNotificationHandler(mock)
	w := httptest.NewRecorder()

	h.GetPreferences(w, newNotifReq(http.MethodGet, "/api/notification-preferences", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.NotificationPreferences
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.UserID != 42 {
		t.Errorf("expected userID 42, got %d", resp.UserID)
	}
}

// --- UpdatePreferences ---

func TestNotificationHandler_UpdatePreferences_Returns200(t *testing.T) {
	mock := &mockNotificationService{
		updatePreferencesFn: func(userID int64, req models.UpdateNotificationPreferencesRequest) error {
			return nil
		},
	}
	h := NewNotificationHandler(mock)

	body, _ := json.Marshal(models.UpdateNotificationPreferencesRequest{})
	w := httptest.NewRecorder()

	h.UpdatePreferences(w, newNotifReq(http.MethodPut, "/api/notification-preferences", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNotificationHandler_UpdatePreferences_InvalidBody_Returns400(t *testing.T) {
	h := NewNotificationHandler(&mockNotificationService{})
	w := httptest.NewRecorder()

	r := newNotifReq(http.MethodPut, "/api/notification-preferences", []byte("{bad json"), 42)
	h.UpdatePreferences(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
