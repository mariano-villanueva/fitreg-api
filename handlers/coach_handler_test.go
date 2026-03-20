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

type mockCoachService struct {
	listStudentsFn               func(coachID int64) ([]models.CoachStudentInfo, error)
	endRelationshipFn            func(csID, userID int64) error
	getStudentWorkoutsFn         func(coachID, studentID int64) ([]models.Workout, error)
	listAssignedWorkoutsFn       func(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error)
	createAssignedWorkoutFn      func(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error)
	getAssignedWorkoutFn         func(awID, coachID int64) (models.AssignedWorkout, error)
	updateAssignedWorkoutFn      func(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error)
	deleteAssignedWorkoutFn      func(awID, coachID int64) error
	getMyAssignedWorkoutsFn      func(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error)
	updateAssignedWorkoutStatusFn func(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) error
	getDailySummaryFn            func(coachID int64, date string) ([]models.DailySummaryItem, error)
	getStudentLoadFn             func(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
	getMyLoadFn                  func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error)
}

func (m *mockCoachService) ListStudents(coachID int64) ([]models.CoachStudentInfo, error) {
	return m.listStudentsFn(coachID)
}
func (m *mockCoachService) EndRelationship(csID, userID int64) error {
	return m.endRelationshipFn(csID, userID)
}
func (m *mockCoachService) GetStudentWorkouts(coachID, studentID int64) ([]models.Workout, error) {
	return m.getStudentWorkoutsFn(coachID, studentID)
}
func (m *mockCoachService) ListAssignedWorkouts(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
	return m.listAssignedWorkoutsFn(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
}
func (m *mockCoachService) CreateAssignedWorkout(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
	return m.createAssignedWorkoutFn(coachID, req)
}
func (m *mockCoachService) GetAssignedWorkout(awID, coachID int64) (models.AssignedWorkout, error) {
	return m.getAssignedWorkoutFn(awID, coachID)
}
func (m *mockCoachService) UpdateAssignedWorkout(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
	return m.updateAssignedWorkoutFn(awID, coachID, req)
}
func (m *mockCoachService) DeleteAssignedWorkout(awID, coachID int64) error {
	return m.deleteAssignedWorkoutFn(awID, coachID)
}
func (m *mockCoachService) GetMyAssignedWorkouts(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
	return m.getMyAssignedWorkoutsFn(studentID, startDate, endDate)
}
func (m *mockCoachService) UpdateAssignedWorkoutStatus(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) error {
	return m.updateAssignedWorkoutStatusFn(awID, studentID, req)
}
func (m *mockCoachService) GetDailySummary(coachID int64, date string, includeSegments bool) ([]models.DailySummaryItem, error) {
	return m.getDailySummaryFn(coachID, date)
}
func (m *mockCoachService) GetStudentLoad(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	return m.getStudentLoadFn(coachID, studentID, weeks)
}
func (m *mockCoachService) GetMyLoad(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
	return m.getMyLoadFn(studentID, weeks)
}

func newCoachReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- ListStudents ---

func TestCoachHandler_ListStudents_ReturnsStudents(t *testing.T) {
	mock := &mockCoachService{
		listStudentsFn: func(coachID int64) ([]models.CoachStudentInfo, error) {
			return []models.CoachStudentInfo{{ID: 1, Name: "Alice"}}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.ListStudents(w, newCoachReq(http.MethodGet, "/api/coach/students", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.CoachStudentInfo
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Alice" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestCoachHandler_ListStudents_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		listStudentsFn: func(coachID int64) ([]models.CoachStudentInfo, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.ListStudents(w, newCoachReq(http.MethodGet, "/api/coach/students", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- EndRelationship ---

func TestCoachHandler_EndRelationship_Returns200(t *testing.T) {
	mock := &mockCoachService{
		endRelationshipFn: func(csID, userID int64) error { return nil },
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.EndRelationship(w, newCoachReq(http.MethodPut, "/api/coach-students/5/end", nil, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCoachHandler_EndRelationship_InvalidID_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})
	w := httptest.NewRecorder()

	h.EndRelationship(w, newCoachReq(http.MethodPut, "/api/coach-students/abc/end", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_EndRelationship_NotFound_Returns404(t *testing.T) {
	mock := &mockCoachService{
		endRelationshipFn: func(csID, userID int64) error { return services.ErrNotFound },
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.EndRelationship(w, newCoachReq(http.MethodPut, "/api/coach-students/99/end", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- GetStudentWorkouts ---

func TestCoachHandler_GetStudentWorkouts_ReturnsWorkouts(t *testing.T) {
	mock := &mockCoachService{
		getStudentWorkoutsFn: func(coachID, studentID int64) ([]models.Workout, error) {
			return []models.Workout{{ID: 7, UserID: studentID}}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetStudentWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/students/3/workouts", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.Workout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != 7 {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestCoachHandler_GetStudentWorkouts_InvalidID_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})
	w := httptest.NewRecorder()

	h.GetStudentWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/students/abc/workouts", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_GetStudentWorkouts_NotCoach_Returns403(t *testing.T) {
	mock := &mockCoachService{
		getStudentWorkoutsFn: func(coachID, studentID int64) ([]models.Workout, error) {
			return nil, services.ErrNotCoach
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetStudentWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/students/3/workouts", nil, 10))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- CreateAssignedWorkout ---

func TestCoachHandler_CreateAssignedWorkout_ReturnsCreated(t *testing.T) {
	mock := &mockCoachService{
		createAssignedWorkoutFn: func(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{ID: 20, Title: req.Title}, nil
		},
	}
	h := NewCoachHandler(mock)

	body, _ := json.Marshal(models.CreateAssignedWorkoutRequest{
		Title:    "Morning Run",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateAssignedWorkout(w, newCoachReq(http.MethodPost, "/api/coach/assigned-workouts", body, 10))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.AssignedWorkout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 20 {
		t.Errorf("expected ID 20, got %d", resp.ID)
	}
}

func TestCoachHandler_CreateAssignedWorkout_MissingTitle_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})

	body, _ := json.Marshal(models.CreateAssignedWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateAssignedWorkout(w, newCoachReq(http.MethodPost, "/api/coach/assigned-workouts", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_CreateAssignedWorkout_NoSegments_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})

	body, _ := json.Marshal(models.CreateAssignedWorkoutRequest{Title: "Run"})
	w := httptest.NewRecorder()

	h.CreateAssignedWorkout(w, newCoachReq(http.MethodPost, "/api/coach/assigned-workouts", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_CreateAssignedWorkout_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		createAssignedWorkoutFn: func(coachID int64, req models.CreateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{}, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)

	body, _ := json.Marshal(models.CreateAssignedWorkoutRequest{
		Title:    "Run",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.CreateAssignedWorkout(w, newCoachReq(http.MethodPost, "/api/coach/assigned-workouts", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetAssignedWorkout ---

func TestCoachHandler_GetAssignedWorkout_ReturnsWorkout(t *testing.T) {
	mock := &mockCoachService{
		getAssignedWorkoutFn: func(awID, coachID int64) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{ID: awID, Title: "Run"}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetAssignedWorkout(w, newCoachReq(http.MethodGet, "/api/coach/assigned-workouts/15", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.AssignedWorkout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 15 {
		t.Errorf("expected ID 15, got %d", resp.ID)
	}
}

func TestCoachHandler_GetAssignedWorkout_NotFound_Returns404(t *testing.T) {
	mock := &mockCoachService{
		getAssignedWorkoutFn: func(awID, coachID int64) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{}, services.ErrNotFound
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetAssignedWorkout(w, newCoachReq(http.MethodGet, "/api/coach/assigned-workouts/99", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- DeleteAssignedWorkout ---

func TestCoachHandler_DeleteAssignedWorkout_Returns200(t *testing.T) {
	mock := &mockCoachService{
		deleteAssignedWorkoutFn: func(awID, coachID int64) error { return nil },
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteAssignedWorkout(w, newCoachReq(http.MethodDelete, "/api/coach/assigned-workouts/5", nil, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCoachHandler_DeleteAssignedWorkout_Finished_Returns409(t *testing.T) {
	mock := &mockCoachService{
		deleteAssignedWorkoutFn: func(awID, coachID int64) error { return services.ErrWorkoutFinished },
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.DeleteAssignedWorkout(w, newCoachReq(http.MethodDelete, "/api/coach/assigned-workouts/5", nil, 10))

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- GetMyAssignedWorkouts ---

func TestCoachHandler_GetMyAssignedWorkouts_ReturnsWorkouts(t *testing.T) {
	mock := &mockCoachService{
		getMyAssignedWorkoutsFn: func(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
			return []models.AssignedWorkout{{ID: 1, Title: "My Run"}}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetMyAssignedWorkouts(w, newCoachReq(http.MethodGet, "/api/my-assigned-workouts", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.AssignedWorkout
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Title != "My Run" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestCoachHandler_GetMyAssignedWorkouts_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		getMyAssignedWorkoutsFn: func(studentID int64, startDate, endDate string) ([]models.AssignedWorkout, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetMyAssignedWorkouts(w, newCoachReq(http.MethodGet, "/api/my-assigned-workouts", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateAssignedWorkoutStatus ---

func TestCoachHandler_UpdateAssignedWorkoutStatus_Completed_Returns200(t *testing.T) {
	feeling := 8
	mock := &mockCoachService{
		updateAssignedWorkoutStatusFn: func(awID, studentID int64, req models.UpdateAssignedWorkoutStatusRequest) error {
			return nil
		},
	}
	h := NewCoachHandler(mock)

	body, _ := json.Marshal(models.UpdateAssignedWorkoutStatusRequest{
		Status:        "completed",
		ResultFeeling: &feeling,
	})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkoutStatus(w, newCoachReq(http.MethodPut, "/api/my-assigned-workouts/3/status", body, 42))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCoachHandler_UpdateAssignedWorkoutStatus_InvalidStatus_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})

	body, _ := json.Marshal(models.UpdateAssignedWorkoutStatusRequest{Status: "invalid"})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkoutStatus(w, newCoachReq(http.MethodPut, "/api/my-assigned-workouts/3/status", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_UpdateAssignedWorkoutStatus_CompletedMissingFeeling_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})

	body, _ := json.Marshal(models.UpdateAssignedWorkoutStatusRequest{Status: "completed"})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkoutStatus(w, newCoachReq(http.MethodPut, "/api/my-assigned-workouts/3/status", body, 42))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GetDailySummary ---

func TestCoachHandler_GetDailySummary_ReturnsItems(t *testing.T) {
	mock := &mockCoachService{
		getDailySummaryFn: func(coachID int64, date string) ([]models.DailySummaryItem, error) {
			return []models.DailySummaryItem{{StudentName: "Bob"}}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetDailySummary(w, newCoachReq(http.MethodGet, "/api/coach/daily-summary?date=2024-03-01", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCoachHandler_GetDailySummary_InvalidDate_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})
	w := httptest.NewRecorder()

	h.GetDailySummary(w, newCoachReq(http.MethodGet, "/api/coach/daily-summary?date=notadate", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_GetDailySummary_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		getDailySummaryFn: func(coachID int64, date string) ([]models.DailySummaryItem, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetDailySummary(w, newCoachReq(http.MethodGet, "/api/coach/daily-summary?date=2024-03-01", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- ListAssignedWorkouts ---

func TestCoachHandler_ListAssignedWorkouts_ReturnsWorkouts(t *testing.T) {
	mock := &mockCoachService{
		listAssignedWorkoutsFn: func(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
			return []models.AssignedWorkout{{ID: 1, Title: "Run"}}, 1, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.ListAssignedWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/assigned-workouts", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCoachHandler_ListAssignedWorkouts_WithPagination_ReturnsPaginatedResponse(t *testing.T) {
	mock := &mockCoachService{
		listAssignedWorkoutsFn: func(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
			return []models.AssignedWorkout{{ID: 2, Title: "Swim"}}, 5, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.ListAssignedWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/assigned-workouts?limit=10&page=1", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["total"]; !ok {
		t.Errorf("expected 'total' field in paginated response")
	}
}

func TestCoachHandler_ListAssignedWorkouts_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		listAssignedWorkoutsFn: func(coachID, studentID int64, statusFilter, startDate, endDate string, limit, offset int) ([]models.AssignedWorkout, int, error) {
			return nil, 0, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.ListAssignedWorkouts(w, newCoachReq(http.MethodGet, "/api/coach/assigned-workouts", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- UpdateAssignedWorkout ---

func TestCoachHandler_UpdateAssignedWorkout_Returns200(t *testing.T) {
	mock := &mockCoachService{
		updateAssignedWorkoutFn: func(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{ID: awID, Title: req.Title}, nil
		},
	}
	h := NewCoachHandler(mock)

	body, _ := json.Marshal(models.UpdateAssignedWorkoutRequest{
		Title:    "Updated Run",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkout(w, newCoachReq(http.MethodPut, "/api/coach/assigned-workouts/5", body, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCoachHandler_UpdateAssignedWorkout_NoSegments_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})

	body, _ := json.Marshal(models.UpdateAssignedWorkoutRequest{Title: "Run"})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkout(w, newCoachReq(http.MethodPut, "/api/coach/assigned-workouts/5", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_UpdateAssignedWorkout_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		updateAssignedWorkoutFn: func(awID, coachID int64, req models.UpdateAssignedWorkoutRequest) (models.AssignedWorkout, error) {
			return models.AssignedWorkout{}, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)

	body, _ := json.Marshal(models.UpdateAssignedWorkoutRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.UpdateAssignedWorkout(w, newCoachReq(http.MethodPut, "/api/coach/assigned-workouts/5", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetStudentLoad ---

func TestCoachHandler_GetStudentLoad_ReturnsLoad(t *testing.T) {
	mock := &mockCoachService{
		getStudentLoadFn: func(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return []models.WeeklyLoadEntry{
				{WeekStart: "2026-03-16", PlannedKm: 42.5, ActualKm: 38.0, SessionsPlanned: 5, SessionsCompleted: 4},
			}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetStudentLoad(w, newCoachReq(http.MethodGet, "/api/coach/students/7/load?weeks=4", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []models.WeeklyLoadEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].PlannedKm != 42.5 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestCoachHandler_GetStudentLoad_InvalidID_Returns400(t *testing.T) {
	h := NewCoachHandler(&mockCoachService{})
	w := httptest.NewRecorder()

	h.GetStudentLoad(w, newCoachReq(http.MethodGet, "/api/coach/students/abc/load", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCoachHandler_GetStudentLoad_Forbidden_Returns403(t *testing.T) {
	mock := &mockCoachService{
		getStudentLoadFn: func(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return nil, services.ErrForbidden
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetStudentLoad(w, newCoachReq(http.MethodGet, "/api/coach/students/99/load", nil, 10))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCoachHandler_GetStudentLoad_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		getStudentLoadFn: func(coachID, studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetStudentLoad(w, newCoachReq(http.MethodGet, "/api/coach/students/7/load", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetMyLoad ---

func TestCoachHandler_GetMyLoad_ReturnsLoad(t *testing.T) {
	mock := &mockCoachService{
		getMyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return []models.WeeklyLoadEntry{
				{WeekStart: "2026-03-16", PlannedKm: 30.0, ActualKm: 30.0, SessionsPlanned: 4, SessionsCompleted: 4},
			}, nil
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetMyLoad(w, newCoachReq(http.MethodGet, "/api/me/load?weeks=8", nil, 42))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []models.WeeklyLoadEntry
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].PlannedKm != 30.0 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestCoachHandler_GetMyLoad_ServiceError_Returns500(t *testing.T) {
	mock := &mockCoachService{
		getMyLoadFn: func(studentID int64, weeks int) ([]models.WeeklyLoadEntry, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCoachHandler(mock)
	w := httptest.NewRecorder()

	h.GetMyLoad(w, newCoachReq(http.MethodGet, "/api/me/load", nil, 42))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
