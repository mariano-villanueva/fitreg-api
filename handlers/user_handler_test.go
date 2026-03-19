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
)

// mockUserService is a test double for UserServicer.
type mockUserService struct {
	getProfileFn            func(userID int64) (*models.UserProfile, error)
	updateProfileFn         func(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error)
	isCoachFn               func(userID int64) (bool, error)
	hasPendingCoachReqFn    func(userID int64) (bool, error)
	setCoachLocalityFn      func(userID int64, locality, level string) error
	getNameAndAvatarFn      func(userID int64) (string, string, error)
	getAdminIDsFn           func() ([]int64, error)
	uploadAvatarFn          func(userID int64, image string) error
	deleteAvatarFn          func(userID int64) error
	getCoachRequestStatusFn func(userID int64) (string, error)
}

func (m *mockUserService) GetProfile(userID int64) (*models.UserProfile, error) {
	return m.getProfileFn(userID)
}
func (m *mockUserService) UpdateProfile(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error) {
	return m.updateProfileFn(userID, req)
}
func (m *mockUserService) IsCoach(userID int64) (bool, error) {
	if m.isCoachFn != nil {
		return m.isCoachFn(userID)
	}
	return false, nil
}
func (m *mockUserService) HasPendingCoachRequest(userID int64) (bool, error) {
	if m.hasPendingCoachReqFn != nil {
		return m.hasPendingCoachReqFn(userID)
	}
	return false, nil
}
func (m *mockUserService) SetCoachLocality(userID int64, locality, level string) error {
	if m.setCoachLocalityFn != nil {
		return m.setCoachLocalityFn(userID, locality, level)
	}
	return nil
}
func (m *mockUserService) GetNameAndAvatar(userID int64) (string, string, error) {
	if m.getNameAndAvatarFn != nil {
		return m.getNameAndAvatarFn(userID)
	}
	return "", "", nil
}
func (m *mockUserService) GetAdminIDs() ([]int64, error) {
	if m.getAdminIDsFn != nil {
		return m.getAdminIDsFn()
	}
	return []int64{}, nil
}
func (m *mockUserService) UploadAvatar(userID int64, image string) error {
	return m.uploadAvatarFn(userID, image)
}
func (m *mockUserService) DeleteAvatar(userID int64) error {
	return m.deleteAvatarFn(userID)
}
func (m *mockUserService) GetCoachRequestStatus(userID int64) (string, error) {
	return m.getCoachRequestStatusFn(userID)
}

// mockNotificationCreator is a test double for NotificationCreator.
type mockNotificationCreator struct {
	createFn func(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error
}

func (m *mockNotificationCreator) Create(userID int64, notifType, title, body string, metadata interface{}, actions []models.NotificationAction) error {
	if m.createFn != nil {
		return m.createFn(userID, notifType, title, body, metadata, actions)
	}
	return nil
}

// newUserReq builds an HTTP request with the given user ID injected in context.
func newUserReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// stubNotifSvc returns a no-op notification mock.
func stubNotifSvc() *mockNotificationCreator {
	return &mockNotificationCreator{}
}

// --- GetProfile ---

func TestUserHandler_GetProfile_ReturnsProfile(t *testing.T) {
	mock := &mockUserService{
		getProfileFn: func(userID int64) (*models.UserProfile, error) {
			return &models.UserProfile{ID: userID, Name: "Test User"}, nil
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.GetProfile(w, newUserReq(http.MethodGet, "/api/me", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 42 || resp.Name != "Test User" {
		t.Errorf("unexpected profile: %+v", resp)
	}
}

func TestUserHandler_GetProfile_ServiceError_Returns500(t *testing.T) {
	mock := &mockUserService{
		getProfileFn: func(userID int64) (*models.UserProfile, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.GetProfile(w, newUserReq(http.MethodGet, "/api/me", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateProfile ---

func TestUserHandler_UpdateProfile_ReturnsUpdated(t *testing.T) {
	mock := &mockUserService{
		updateProfileFn: func(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error) {
			return &models.UserProfile{ID: userID, Name: req.Name}, nil
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(models.UpdateProfileRequest{Name: "Updated"})
	w := httptest.NewRecorder()

	h.UpdateProfile(w, newUserReq(http.MethodPut, "/api/me", body, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", resp.Name)
	}
}

func TestUserHandler_UpdateProfile_InvalidJSON_Returns400(t *testing.T) {
	h := NewUserHandler(&mockUserService{}, stubNotifSvc())
	w := httptest.NewRecorder()

	r := newUserReq(http.MethodPut, "/api/me", []byte("{bad json"), 42)
	h.UpdateProfile(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserHandler_UpdateProfile_ServiceError_Returns500(t *testing.T) {
	mock := &mockUserService{
		updateProfileFn: func(userID int64, req models.UpdateProfileRequest) (*models.UserProfile, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(models.UpdateProfileRequest{Name: "Name"})
	w := httptest.NewRecorder()

	h.UpdateProfile(w, newUserReq(http.MethodPut, "/api/me", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UploadAvatar ---

// validAvatarDataURI is a 1×1 transparent PNG as a base64 data URI (< 500KB).
const validAvatarDataURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

func TestUserHandler_UploadAvatar_Success_Returns200(t *testing.T) {
	mock := &mockUserService{
		uploadAvatarFn: func(userID int64, image string) error { return nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]string{"image": validAvatarDataURI})
	w := httptest.NewRecorder()

	h.UploadAvatar(w, newUserReq(http.MethodPost, "/api/me/avatar", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UploadAvatar_MissingImage_Returns400(t *testing.T) {
	h := NewUserHandler(&mockUserService{}, stubNotifSvc())

	body, _ := json.Marshal(map[string]string{"image": ""})
	w := httptest.NewRecorder()

	h.UploadAvatar(w, newUserReq(http.MethodPost, "/api/me/avatar", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserHandler_UploadAvatar_InvalidPrefix_Returns400(t *testing.T) {
	h := NewUserHandler(&mockUserService{}, stubNotifSvc())

	body, _ := json.Marshal(map[string]string{"image": "notadatauri"})
	w := httptest.NewRecorder()

	h.UploadAvatar(w, newUserReq(http.MethodPost, "/api/me/avatar", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserHandler_UploadAvatar_InvalidBase64_Returns400(t *testing.T) {
	h := NewUserHandler(&mockUserService{}, stubNotifSvc())

	body, _ := json.Marshal(map[string]string{"image": "data:image/png;base64,!!!notvalidbase64!!!"})
	w := httptest.NewRecorder()

	h.UploadAvatar(w, newUserReq(http.MethodPost, "/api/me/avatar", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserHandler_UploadAvatar_ServiceError_Returns500(t *testing.T) {
	mock := &mockUserService{
		uploadAvatarFn: func(userID int64, image string) error { return errors.New("storage error") },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]string{"image": validAvatarDataURI})
	w := httptest.NewRecorder()

	h.UploadAvatar(w, newUserReq(http.MethodPost, "/api/me/avatar", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- DeleteAvatar ---

func TestUserHandler_DeleteAvatar_Success_Returns200(t *testing.T) {
	mock := &mockUserService{
		deleteAvatarFn: func(userID int64) error { return nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.DeleteAvatar(w, newUserReq(http.MethodDelete, "/api/me/avatar", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUserHandler_DeleteAvatar_ServiceError_Returns500(t *testing.T) {
	mock := &mockUserService{
		deleteAvatarFn: func(userID int64) error { return errors.New("db error") },
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.DeleteAvatar(w, newUserReq(http.MethodDelete, "/api/me/avatar", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetCoachRequestStatus ---

func TestUserHandler_GetCoachRequestStatus_ReturnsStatus(t *testing.T) {
	mock := &mockUserService{
		getCoachRequestStatusFn: func(userID int64) (string, error) {
			return "pending", nil
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.GetCoachRequestStatus(w, newUserReq(http.MethodGet, "/api/me/coach-request-status", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "pending" {
		t.Errorf("expected 'pending', got %q", resp["status"])
	}
}

func TestUserHandler_GetCoachRequestStatus_ServiceError_Returns500(t *testing.T) {
	mock := &mockUserService{
		getCoachRequestStatusFn: func(userID int64) (string, error) {
			return "", errors.New("db error")
		},
	}
	h := NewUserHandler(mock, stubNotifSvc())
	w := httptest.NewRecorder()

	h.GetCoachRequestStatus(w, newUserReq(http.MethodGet, "/api/me/coach-request-status", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- RequestCoach ---

func TestUserHandler_RequestCoach_Returns200(t *testing.T) {
	mock := &mockUserService{
		isCoachFn:            func(userID int64) (bool, error) { return false, nil },
		hasPendingCoachReqFn: func(userID int64) (bool, error) { return false, nil },
		setCoachLocalityFn:   func(userID int64, locality, level string) error { return nil },
		getNameAndAvatarFn:   func(userID int64) (string, string, error) { return "Alice", "", nil },
		getAdminIDsFn:        func() ([]int64, error) { return []int64{1}, nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]interface{}{"locality": "Buenos Aires", "level": []string{"beginner"}})
	w := httptest.NewRecorder()

	h.RequestCoach(w, newUserReq(http.MethodPost, "/api/me/request-coach", body, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_RequestCoach_MissingLevel_Returns400(t *testing.T) {
	h := NewUserHandler(&mockUserService{}, stubNotifSvc())

	body, _ := json.Marshal(map[string]interface{}{"locality": "BA", "level": []string{}})
	w := httptest.NewRecorder()

	h.RequestCoach(w, newUserReq(http.MethodPost, "/api/me/request-coach", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserHandler_RequestCoach_AlreadyCoach_Returns409(t *testing.T) {
	mock := &mockUserService{
		isCoachFn: func(userID int64) (bool, error) { return true, nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]interface{}{"level": []string{"beginner"}})
	w := httptest.NewRecorder()

	h.RequestCoach(w, newUserReq(http.MethodPost, "/api/me/request-coach", body, 42))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestUserHandler_RequestCoach_AlreadyPending_Returns409(t *testing.T) {
	mock := &mockUserService{
		isCoachFn:            func(userID int64) (bool, error) { return false, nil },
		hasPendingCoachReqFn: func(userID int64) (bool, error) { return true, nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]interface{}{"level": []string{"beginner"}})
	w := httptest.NewRecorder()

	h.RequestCoach(w, newUserReq(http.MethodPost, "/api/me/request-coach", body, 42))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestUserHandler_RequestCoach_NoAdmins_Returns200(t *testing.T) {
	mock := &mockUserService{
		isCoachFn:            func(userID int64) (bool, error) { return false, nil },
		hasPendingCoachReqFn: func(userID int64) (bool, error) { return false, nil },
		setCoachLocalityFn:   func(userID int64, locality, level string) error { return nil },
		getNameAndAvatarFn:   func(userID int64) (string, string, error) { return "Bob", "", nil },
		getAdminIDsFn:        func() ([]int64, error) { return []int64{}, nil },
	}
	h := NewUserHandler(mock, stubNotifSvc())

	body, _ := json.Marshal(map[string]interface{}{"level": []string{"advanced"}})
	w := httptest.NewRecorder()

	h.RequestCoach(w, newUserReq(http.MethodPost, "/api/me/request-coach", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
