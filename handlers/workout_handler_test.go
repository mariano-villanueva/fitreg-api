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

// mockWorkoutService is a test double for WorkoutServicer.
type mockWorkoutService struct {
	listFn    func(userID int64, startDate, endDate string) ([]models.Workout, error)
	getByIDFn func(id, userID int64) (models.Workout, error)
	createFn  func(userID int64, req models.CreateWorkoutRequest) (models.Workout, error)
	updateFn  func(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error)
	deleteFn  func(id, userID int64) error
}

func (m *mockWorkoutService) List(userID int64, startDate, endDate string) ([]models.Workout, error) {
	if m.listFn != nil {
		return m.listFn(userID, startDate, endDate)
	}
	return nil, nil
}
func (m *mockWorkoutService) GetByID(id, userID int64) (models.Workout, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id, userID)
	}
	return models.Workout{}, nil
}
func (m *mockWorkoutService) Create(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
	if m.createFn != nil {
		return m.createFn(userID, req)
	}
	return models.Workout{}, nil
}
func (m *mockWorkoutService) Update(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
	if m.updateFn != nil {
		return m.updateFn(id, userID, req)
	}
	return models.Workout{}, nil
}
func (m *mockWorkoutService) Delete(id, userID int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(id, userID)
	}
	return nil
}
func (m *mockWorkoutService) UpdateStatus(id, userID int64, req models.UpdateWorkoutStatusRequest) error {
	return nil
}
func (m *mockWorkoutService) GetMyWorkouts(studentID int64, startDate, endDate string) ([]models.Workout, error) {
	return nil, nil
}
func (m *mockWorkoutService) CreateCoachWorkout(coachID int64, req models.CreateCoachWorkoutRequest) (models.Workout, error) {
	return models.Workout{}, nil
}
func (m *mockWorkoutService) ListCoachWorkouts(coachID int64, studentID *int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.Workout, int, error) {
	return nil, 0, nil
}
func (m *mockWorkoutService) GetCoachWorkout(workoutID, coachID int64) (models.Workout, error) {
	return models.Workout{}, nil
}
func (m *mockWorkoutService) UpdateCoachWorkout(workoutID, coachID int64, req models.UpdateCoachWorkoutRequest) (models.Workout, error) {
	return models.Workout{}, nil
}
func (m *mockWorkoutService) DeleteCoachWorkout(workoutID, coachID int64) error {
	return nil
}

// newWorkoutReq builds an HTTP request with the given user ID injected in context.
func newWorkoutReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- ListWorkouts ---

func TestWorkoutHandler_List_ReturnsWorkouts(t *testing.T) {
	mock := &mockWorkoutService{
		listFn: func(userID int64, startDate, endDate string) ([]models.Workout, error) {
			return []models.Workout{{ID: 1, UserID: userID, DueDate: "2024-01-01"}}, nil
		},
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.ListWorkouts(w, newWorkoutReq(http.MethodGet, "/api/workouts", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != 1 {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestWorkoutHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockWorkoutService{
		listFn: func(userID int64, startDate, endDate string) ([]models.Workout, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.ListWorkouts(w, newWorkoutReq(http.MethodGet, "/api/workouts", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetWorkout ---

func TestWorkoutHandler_Get_ReturnsWorkout(t *testing.T) {
	mock := &mockWorkoutService{
		getByIDFn: func(id, userID int64) (models.Workout, error) {
			return models.Workout{ID: id, UserID: userID, DueDate: "2024-01-01"}, nil
		},
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.GetWorkout(w, newWorkoutReq(http.MethodGet, "/api/workouts/7", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 7 {
		t.Errorf("expected ID 7, got %d", resp.ID)
	}
}

func TestWorkoutHandler_Get_InvalidID_Returns400(t *testing.T) {
	h := NewWorkoutHandler(&mockWorkoutService{})
	w := httptest.NewRecorder()

	h.GetWorkout(w, newWorkoutReq(http.MethodGet, "/api/workouts/abc", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_Get_NotFound_Returns404(t *testing.T) {
	mock := &mockWorkoutService{
		getByIDFn: func(id, userID int64) (models.Workout, error) {
			return models.Workout{}, services.ErrNotFound
		},
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.GetWorkout(w, newWorkoutReq(http.MethodGet, "/api/workouts/99", nil, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- CreateWorkout ---

func TestWorkoutHandler_Create_ReturnsCreated(t *testing.T) {
	mock := &mockWorkoutService{
		createFn: func(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
			return models.Workout{ID: 10, UserID: userID, DueDate: req.DueDate}, nil
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.CreateWorkoutRequest{
		DueDate:  "2024-03-01",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateWorkout(w, newWorkoutReq(http.MethodPost, "/api/workouts", body, 42))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 10 {
		t.Errorf("expected ID 10, got %d", resp.ID)
	}
}

func TestWorkoutHandler_Create_MissingDate_Returns400(t *testing.T) {
	h := NewWorkoutHandler(&mockWorkoutService{})

	body, _ := json.Marshal(models.CreateWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateWorkout(w, newWorkoutReq(http.MethodPost, "/api/workouts", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_Create_NoSegments_ReturnsCreated(t *testing.T) {
	mock := &mockWorkoutService{
		createFn: func(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
			return models.Workout{ID: 10, UserID: userID, DueDate: req.DueDate}, nil
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.CreateWorkoutRequest{DueDate: "2024-03-01"})
	w := httptest.NewRecorder()

	h.CreateWorkout(w, newWorkoutReq(http.MethodPost, "/api/workouts", body, 42))

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestWorkoutHandler_Create_InvalidBody_Returns400(t *testing.T) {
	h := NewWorkoutHandler(&mockWorkoutService{})
	w := httptest.NewRecorder()

	r := newWorkoutReq(http.MethodPost, "/api/workouts", []byte("{bad json"), 42)
	h.CreateWorkout(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_Create_ServiceError_Returns500(t *testing.T) {
	mock := &mockWorkoutService{
		createFn: func(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
			return models.Workout{}, errors.New("db error")
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.CreateWorkoutRequest{
		DueDate:  "2024-03-01",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateWorkout(w, newWorkoutReq(http.MethodPost, "/api/workouts", body, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateWorkout ---

func TestWorkoutHandler_Update_ReturnsWorkout(t *testing.T) {
	mock := &mockWorkoutService{
		updateFn: func(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
			return models.Workout{ID: id, UserID: userID}, nil
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.UpdateWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.UpdateWorkout(w, newWorkoutReq(http.MethodPut, "/api/workouts/5", body, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 5 {
		t.Errorf("expected ID 5, got %d", resp.ID)
	}
}

func TestWorkoutHandler_Update_InvalidID_Returns400(t *testing.T) {
	h := NewWorkoutHandler(&mockWorkoutService{})

	body, _ := json.Marshal(models.UpdateWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.UpdateWorkout(w, newWorkoutReq(http.MethodPut, "/api/workouts/abc", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_Update_NoSegments_ReturnsOK(t *testing.T) {
	mock := &mockWorkoutService{
		updateFn: func(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
			return models.Workout{ID: id, UserID: userID}, nil
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.UpdateWorkoutRequest{})
	w := httptest.NewRecorder()

	h.UpdateWorkout(w, newWorkoutReq(http.MethodPut, "/api/workouts/5", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWorkoutHandler_Update_NotFound_Returns404(t *testing.T) {
	mock := &mockWorkoutService{
		updateFn: func(id, userID int64, req models.UpdateWorkoutRequest) (models.Workout, error) {
			return models.Workout{}, services.ErrNotFound
		},
	}
	h := NewWorkoutHandler(mock)

	body, _ := json.Marshal(models.UpdateWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.UpdateWorkout(w, newWorkoutReq(http.MethodPut, "/api/workouts/99", body, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- DeleteWorkout ---

func TestWorkoutHandler_Delete_Success_Returns200(t *testing.T) {
	mock := &mockWorkoutService{
		deleteFn: func(id, userID int64) error { return nil },
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteWorkout(w, newWorkoutReq(http.MethodDelete, "/api/workouts/3", nil, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWorkoutHandler_Delete_InvalidID_Returns400(t *testing.T) {
	h := NewWorkoutHandler(&mockWorkoutService{})
	w := httptest.NewRecorder()

	h.DeleteWorkout(w, newWorkoutReq(http.MethodDelete, "/api/workouts/abc", nil, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWorkoutHandler_Delete_NotFound_Returns404(t *testing.T) {
	mock := &mockWorkoutService{
		deleteFn: func(id, userID int64) error { return services.ErrNotFound },
	}
	h := NewWorkoutHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteWorkout(w, newWorkoutReq(http.MethodDelete, "/api/workouts/99", nil, 42))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
