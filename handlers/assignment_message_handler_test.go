package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type mockAssignmentMessageService struct {
	listMessagesFn         func(workoutID, userID int64) ([]models.AssignmentMessage, error)
	sendMessageFn          func(workoutID, senderID int64, body string) (models.AssignmentMessage, error)
	markReadFn             func(workoutID, userID int64) error
	getWorkoutDetailFn     func(workoutID, userID int64) (models.Workout, error)
}

func (m *mockAssignmentMessageService) ListMessages(workoutID, userID int64) ([]models.AssignmentMessage, error) {
	return m.listMessagesFn(workoutID, userID)
}
func (m *mockAssignmentMessageService) SendMessage(workoutID, senderID int64, body string) (models.AssignmentMessage, error) {
	return m.sendMessageFn(workoutID, senderID, body)
}
func (m *mockAssignmentMessageService) MarkRead(workoutID, userID int64) error {
	return m.markReadFn(workoutID, userID)
}
func (m *mockAssignmentMessageService) GetWorkoutDetail(workoutID, userID int64) (models.Workout, error) {
	return m.getWorkoutDetailFn(workoutID, userID)
}

func newMsgReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- ListMessages ---

func TestAssignmentMessageHandler_List_ReturnsMessages(t *testing.T) {
	mock := &mockAssignmentMessageService{
		listMessagesFn: func(awID, userID int64) ([]models.AssignmentMessage, error) {
			return []models.AssignmentMessage{{ID: 1, Body: "Good job!"}}, nil
		},
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.ListMessages(w, newMsgReq(http.MethodGet, "/api/assignment-messages/5", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.AssignmentMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Body != "Good job!" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestAssignmentMessageHandler_List_InvalidID_Returns400(t *testing.T) {
	h := NewAssignmentMessageHandler(&mockAssignmentMessageService{})
	w := httptest.NewRecorder()

	h.ListMessages(w, newMsgReq(http.MethodGet, "/api/assignment-messages/abc", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_List_Forbidden_Returns403(t *testing.T) {
	mock := &mockAssignmentMessageService{
		listMessagesFn: func(awID, userID int64) ([]models.AssignmentMessage, error) {
			return nil, services.ErrForbidden
		},
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.ListMessages(w, newMsgReq(http.MethodGet, "/api/assignment-messages/5", nil, 42))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockAssignmentMessageService{
		listMessagesFn: func(awID, userID int64) ([]models.AssignmentMessage, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.ListMessages(w, newMsgReq(http.MethodGet, "/api/assignment-messages/5", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- SendMessage ---

func TestAssignmentMessageHandler_Send_ReturnsCreated(t *testing.T) {
	mock := &mockAssignmentMessageService{
		sendMessageFn: func(awID, senderID int64, body string) (models.AssignmentMessage, error) {
			return models.AssignmentMessage{ID: 10, Body: body}, nil
		},
	}
	h := NewAssignmentMessageHandler(mock)

	body, _ := json.Marshal(models.CreateAssignmentMessageRequest{Body: "Great workout!"})
	w := httptest.NewRecorder()

	h.SendMessage(w, newMsgReq(http.MethodPost, "/api/assignment-messages/5", body, 42))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.AssignmentMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 10 {
		t.Errorf("expected ID 10, got %d", resp.ID)
	}
}

func TestAssignmentMessageHandler_Send_EmptyBody_Returns400(t *testing.T) {
	h := NewAssignmentMessageHandler(&mockAssignmentMessageService{})

	body, _ := json.Marshal(models.CreateAssignmentMessageRequest{Body: "   "})
	w := httptest.NewRecorder()

	h.SendMessage(w, newMsgReq(http.MethodPost, "/api/assignment-messages/5", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_Send_TooLong_Returns400(t *testing.T) {
	h := NewAssignmentMessageHandler(&mockAssignmentMessageService{})

	body, _ := json.Marshal(models.CreateAssignmentMessageRequest{Body: strings.Repeat("x", 2001)})
	w := httptest.NewRecorder()

	h.SendMessage(w, newMsgReq(http.MethodPost, "/api/assignment-messages/5", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_Send_ServiceError_Returns500(t *testing.T) {
	mock := &mockAssignmentMessageService{
		sendMessageFn: func(awID, senderID int64, body string) (models.AssignmentMessage, error) {
			return models.AssignmentMessage{}, errors.New("db error")
		},
	}
	h := NewAssignmentMessageHandler(mock)

	body, _ := json.Marshal(models.CreateAssignmentMessageRequest{Body: "Hello"})
	w := httptest.NewRecorder()

	h.SendMessage(w, newMsgReq(http.MethodPost, "/api/assignment-messages/5", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- MarkRead ---

func TestAssignmentMessageHandler_MarkRead_Returns200(t *testing.T) {
	mock := &mockAssignmentMessageService{
		markReadFn: func(awID, userID int64) error { return nil },
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.MarkRead(w, newMsgReq(http.MethodPut, "/api/assignment-messages/5/read", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_MarkRead_InvalidID_Returns400(t *testing.T) {
	h := NewAssignmentMessageHandler(&mockAssignmentMessageService{})
	w := httptest.NewRecorder()

	h.MarkRead(w, newMsgReq(http.MethodPut, "/api/assignment-messages/abc/read", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAssignmentMessageHandler_MarkRead_ServiceError_Returns500(t *testing.T) {
	mock := &mockAssignmentMessageService{
		markReadFn: func(awID, userID int64) error { return errors.New("db error") },
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.MarkRead(w, newMsgReq(http.MethodPut, "/api/assignment-messages/5/read", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetWorkoutDetail ---

func TestAssignmentMessageHandler_GetDetail_ReturnsWorkout(t *testing.T) {
	mock := &mockAssignmentMessageService{
		getWorkoutDetailFn: func(workoutID, userID int64) (models.Workout, error) {
			return models.Workout{ID: workoutID, Title: "Long Run"}, nil
		},
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.GetAssignedWorkoutDetail(w, newMsgReq(http.MethodGet, "/api/assigned-workout-detail/7", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 7 || resp.Title != "Long Run" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestAssignmentMessageHandler_GetDetail_NotFound_Returns404(t *testing.T) {
	mock := &mockAssignmentMessageService{
		getWorkoutDetailFn: func(workoutID, userID int64) (models.Workout, error) {
			return models.Workout{}, services.ErrNotFound
		},
	}
	h := NewAssignmentMessageHandler(mock)
	w := httptest.NewRecorder()

	h.GetAssignedWorkoutDetail(w, newMsgReq(http.MethodGet, "/api/assigned-workout-detail/99", nil, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
