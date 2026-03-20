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

type mockInvitationService struct {
	createFn  func(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error)
	listFn    func(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error)
	getByIDFn func(invID, requestingUserID int64) (models.Invitation, error)
	respondFn func(invID, userID int64, action string) error
	cancelFn  func(invID, userID int64) error
}

func (m *mockInvitationService) Create(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error) {
	return m.createFn(senderID, req)
}
func (m *mockInvitationService) List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
	return m.listFn(userID, status, direction, limit, offset)
}
func (m *mockInvitationService) GetByID(invID, requestingUserID int64) (models.Invitation, error) {
	return m.getByIDFn(invID, requestingUserID)
}
func (m *mockInvitationService) Respond(invID, userID int64, action string) error {
	return m.respondFn(invID, userID, action)
}
func (m *mockInvitationService) Cancel(invID, userID int64) error {
	return m.cancelFn(invID, userID)
}

func newInvReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- CreateInvitation ---

func TestInvitationHandler_Create_ReturnsCreated(t *testing.T) {
	mock := &mockInvitationService{
		createFn: func(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error) {
			return models.Invitation{ID: 1, SenderID: senderID}, nil
		},
	}
	h := NewInvitationHandler(mock)

	body, _ := json.Marshal(models.CreateInvitationRequest{ReceiverID: 99})
	w := httptest.NewRecorder()

	h.CreateInvitation(w, newInvReq(http.MethodPost, "/api/invitations", body, 42))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp models.Invitation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 1 || resp.SenderID != 42 {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestInvitationHandler_Create_ServiceError_Returns500(t *testing.T) {
	mock := &mockInvitationService{
		createFn: func(senderID int64, req models.CreateInvitationRequest) (models.Invitation, error) {
			return models.Invitation{}, errors.New("db error")
		},
	}
	h := NewInvitationHandler(mock)

	body, _ := json.Marshal(models.CreateInvitationRequest{ReceiverID: 99})
	w := httptest.NewRecorder()

	h.CreateInvitation(w, newInvReq(http.MethodPost, "/api/invitations", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- ListInvitations ---

func TestInvitationHandler_List_ReturnsInvitations(t *testing.T) {
	mock := &mockInvitationService{
		listFn: func(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
			return []models.Invitation{{ID: 1}, {ID: 2}}, nil
		},
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.ListInvitations(w, newInvReq(http.MethodGet, "/api/invitations", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.Invitation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 invitations, got %d", len(resp))
	}
}

func TestInvitationHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockInvitationService{
		listFn: func(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.ListInvitations(w, newInvReq(http.MethodGet, "/api/invitations", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetInvitation ---

func TestInvitationHandler_Get_ReturnsInvitation(t *testing.T) {
	mock := &mockInvitationService{
		getByIDFn: func(invID, requestingUserID int64) (models.Invitation, error) {
			return models.Invitation{ID: invID}, nil
		},
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.GetInvitation(w, newInvReq(http.MethodGet, "/api/invitations/7", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.Invitation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 7 {
		t.Errorf("expected ID 7, got %d", resp.ID)
	}
}

func TestInvitationHandler_Get_InvalidID_Returns400(t *testing.T) {
	h := NewInvitationHandler(&mockInvitationService{})
	w := httptest.NewRecorder()

	h.GetInvitation(w, newInvReq(http.MethodGet, "/api/invitations/abc", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInvitationHandler_Get_NotFound_Returns404(t *testing.T) {
	mock := &mockInvitationService{
		getByIDFn: func(invID, requestingUserID int64) (models.Invitation, error) {
			return models.Invitation{}, services.ErrNotFound
		},
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.GetInvitation(w, newInvReq(http.MethodGet, "/api/invitations/99", nil, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- RespondInvitation ---

func TestInvitationHandler_Respond_Returns200(t *testing.T) {
	mock := &mockInvitationService{
		respondFn: func(invID, userID int64, action string) error { return nil },
	}
	h := NewInvitationHandler(mock)

	body, _ := json.Marshal(models.RespondInvitationRequest{Action: "accept"})
	w := httptest.NewRecorder()

	h.RespondInvitation(w, newInvReq(http.MethodPut, "/api/invitations/5/respond", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInvitationHandler_Respond_NotPending_Returns409(t *testing.T) {
	mock := &mockInvitationService{
		respondFn: func(invID, userID int64, action string) error { return services.ErrInvitationNotPending },
	}
	h := NewInvitationHandler(mock)

	body, _ := json.Marshal(models.RespondInvitationRequest{Action: "accept"})
	w := httptest.NewRecorder()

	h.RespondInvitation(w, newInvReq(http.MethodPut, "/api/invitations/5/respond", body, 42))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestInvitationHandler_Respond_MaxCoaches_Returns409(t *testing.T) {
	mock := &mockInvitationService{
		respondFn: func(invID, userID int64, action string) error { return services.ErrStudentMaxCoaches },
	}
	h := NewInvitationHandler(mock)

	body, _ := json.Marshal(models.RespondInvitationRequest{Action: "accept"})
	w := httptest.NewRecorder()

	h.RespondInvitation(w, newInvReq(http.MethodPut, "/api/invitations/5/respond", body, 42))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- CancelInvitation ---

func TestInvitationHandler_Cancel_Returns200(t *testing.T) {
	mock := &mockInvitationService{
		cancelFn: func(invID, userID int64) error { return nil },
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.CancelInvitation(w, newInvReq(http.MethodDelete, "/api/invitations/5", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInvitationHandler_Cancel_Forbidden_Returns403(t *testing.T) {
	mock := &mockInvitationService{
		cancelFn: func(invID, userID int64) error { return services.ErrForbidden },
	}
	h := NewInvitationHandler(mock)
	w := httptest.NewRecorder()

	h.CancelInvitation(w, newInvReq(http.MethodDelete, "/api/invitations/5", nil, 42))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
