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

type mockWeeklyTemplateService struct {
	listFn       func(coachID int64) ([]models.WeeklyTemplate, error)
	createFn     func(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error)
	getFn        func(id, coachID int64) (models.WeeklyTemplate, error)
	updateMetaFn func(id, coachID int64, req models.UpdateWeeklyTemplateRequest) (models.WeeklyTemplate, error)
	deleteFn     func(id, coachID int64) error
	putDaysFn    func(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error)
	assignFn     func(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error)
}

func (m *mockWeeklyTemplateService) List(coachID int64) ([]models.WeeklyTemplate, error) {
	return m.listFn(coachID)
}
func (m *mockWeeklyTemplateService) Create(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
	return m.createFn(coachID, req)
}
func (m *mockWeeklyTemplateService) Get(id, coachID int64) (models.WeeklyTemplate, error) {
	return m.getFn(id, coachID)
}
func (m *mockWeeklyTemplateService) UpdateMeta(id, coachID int64, req models.UpdateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
	return m.updateMetaFn(id, coachID, req)
}
func (m *mockWeeklyTemplateService) Delete(id, coachID int64) error {
	return m.deleteFn(id, coachID)
}
func (m *mockWeeklyTemplateService) PutDays(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error) {
	return m.putDaysFn(id, coachID, req)
}
func (m *mockWeeklyTemplateService) Assign(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error) {
	return m.assignFn(id, coachID, req)
}

func newWTReq(method, path string, body []byte, userID int64) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	return r.WithContext(middleware.WithUserID(r.Context(), userID))
}

// --- List ---

func TestWeeklyTemplateHandler_List_ReturnsTemplates(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		listFn: func(coachID int64) ([]models.WeeklyTemplate, error) {
			return []models.WeeklyTemplate{{ID: 1, Name: "Week A"}}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.List(w, newWTReq(http.MethodGet, "/api/coach/weekly-templates", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.WeeklyTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "Week A" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestWeeklyTemplateHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		listFn: func(coachID int64) ([]models.WeeklyTemplate, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.List(w, newWTReq(http.MethodGet, "/api/coach/weekly-templates", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Create ---

func TestWeeklyTemplateHandler_Create_ReturnsCreated(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		createFn: func(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{ID: 5, Name: req.Name}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.CreateWeeklyTemplateRequest{Name: "Week B"})
	w := httptest.NewRecorder()

	h.Create(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates", body, 10))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.WeeklyTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 5 || resp.Name != "Week B" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestWeeklyTemplateHandler_Create_InvalidBody_Returns400(t *testing.T) {
	h := NewWeeklyTemplateHandler(&mockWeeklyTemplateService{})
	w := httptest.NewRecorder()

	h.Create(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates", []byte("{bad json"), 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_Create_ServiceError_Returns500(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		createFn: func(coachID int64, req models.CreateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{}, errors.New("db error")
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.CreateWeeklyTemplateRequest{Name: "X"})
	w := httptest.NewRecorder()

	h.Create(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Get ---

func TestWeeklyTemplateHandler_Get_ReturnsTemplate(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		getFn: func(id, coachID int64) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{ID: id, Name: "Week C"}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Get(w, newWTReq(http.MethodGet, "/api/coach/weekly-templates/3", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.WeeklyTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 3 {
		t.Errorf("expected ID 3, got %d", resp.ID)
	}
}

func TestWeeklyTemplateHandler_Get_InvalidID_Returns400(t *testing.T) {
	h := NewWeeklyTemplateHandler(&mockWeeklyTemplateService{})
	w := httptest.NewRecorder()

	h.Get(w, newWTReq(http.MethodGet, "/api/coach/weekly-templates/abc", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_Get_NotFound_Returns404(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		getFn: func(id, coachID int64) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{}, services.ErrNotFound
		},
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Get(w, newWTReq(http.MethodGet, "/api/coach/weekly-templates/99", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- UpdateMeta ---

func TestWeeklyTemplateHandler_UpdateMeta_Returns200(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		updateMetaFn: func(id, coachID int64, req models.UpdateWeeklyTemplateRequest) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{ID: id, Name: req.Name}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.UpdateWeeklyTemplateRequest{Name: "Updated"})
	w := httptest.NewRecorder()

	h.UpdateMeta(w, newWTReq(http.MethodPut, "/api/coach/weekly-templates/3", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_UpdateMeta_InvalidID_Returns400(t *testing.T) {
	h := NewWeeklyTemplateHandler(&mockWeeklyTemplateService{})

	body, _ := json.Marshal(models.UpdateWeeklyTemplateRequest{Name: "X"})
	w := httptest.NewRecorder()

	h.UpdateMeta(w, newWTReq(http.MethodPut, "/api/coach/weekly-templates/abc", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Delete ---

func TestWeeklyTemplateHandler_Delete_Returns204(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		deleteFn: func(id, coachID int64) error { return nil },
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Delete(w, newWTReq(http.MethodDelete, "/api/coach/weekly-templates/3", nil, 10))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_Delete_NotFound_Returns404(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		deleteFn: func(id, coachID int64) error { return services.ErrNotFound },
	}
	h := NewWeeklyTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Delete(w, newWTReq(http.MethodDelete, "/api/coach/weekly-templates/99", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- PutDays ---

func TestWeeklyTemplateHandler_PutDays_Returns200(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		putDaysFn: func(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{ID: id}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.PutDaysRequest{Days: []models.WeeklyTemplateDayRequest{}})
	w := httptest.NewRecorder()

	h.PutDays(w, newWTReq(http.MethodPut, "/api/coach/weekly-templates/3/days", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_PutDays_NilDays_DefaultsToEmpty(t *testing.T) {
	var capturedReq models.PutDaysRequest
	mock := &mockWeeklyTemplateService{
		putDaysFn: func(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error) {
			capturedReq = req
			return models.WeeklyTemplate{ID: id}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	// Send body with null days
	body, _ := json.Marshal(map[string]interface{}{"days": nil})
	w := httptest.NewRecorder()

	h.PutDays(w, newWTReq(http.MethodPut, "/api/coach/weekly-templates/3/days", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedReq.Days == nil {
		t.Error("expected Days to be defaulted to empty slice, got nil")
	}
}

func TestWeeklyTemplateHandler_PutDays_ServiceError_Returns500(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		putDaysFn: func(id, coachID int64, req models.PutDaysRequest) (models.WeeklyTemplate, error) {
			return models.WeeklyTemplate{}, errors.New("db error")
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.PutDaysRequest{})
	w := httptest.NewRecorder()

	h.PutDays(w, newWTReq(http.MethodPut, "/api/coach/weekly-templates/3/days", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Assign ---

func TestWeeklyTemplateHandler_Assign_ReturnsCreated(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		assignFn: func(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error) {
			return models.AssignWeeklyTemplateResponse{AssignedWorkoutIDs: []int64{1, 2, 3}}, nil
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.AssignWeeklyTemplateRequest{
		StudentID: 99,
		StartDate: "2024-03-04",
	})
	w := httptest.NewRecorder()

	h.Assign(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates/3/assign", body, 10))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.AssignWeeklyTemplateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.AssignedWorkoutIDs) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(resp.AssignedWorkoutIDs))
	}
}

func TestWeeklyTemplateHandler_Assign_MissingStudentID_Returns400(t *testing.T) {
	h := NewWeeklyTemplateHandler(&mockWeeklyTemplateService{})

	body, _ := json.Marshal(models.AssignWeeklyTemplateRequest{StartDate: "2024-03-04"})
	w := httptest.NewRecorder()

	h.Assign(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates/3/assign", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_Assign_MissingStartDate_Returns400(t *testing.T) {
	h := NewWeeklyTemplateHandler(&mockWeeklyTemplateService{})

	body, _ := json.Marshal(models.AssignWeeklyTemplateRequest{StudentID: 99})
	w := httptest.NewRecorder()

	h.Assign(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates/3/assign", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWeeklyTemplateHandler_Assign_Conflict_Returns409(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		assignFn: func(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error) {
			return models.AssignWeeklyTemplateResponse{}, &services.ConflictError{Dates: []string{"2024-03-04", "2024-03-05"}}
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.AssignWeeklyTemplateRequest{StudentID: 99, StartDate: "2024-03-04"})
	w := httptest.NewRecorder()

	h.Assign(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates/3/assign", body, 10))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var resp models.AssignConflictResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.ConflictingDates) != 2 {
		t.Errorf("expected 2 conflicting dates, got %d", len(resp.ConflictingDates))
	}
}

func TestWeeklyTemplateHandler_Assign_ServiceError_Returns500(t *testing.T) {
	mock := &mockWeeklyTemplateService{
		assignFn: func(id, coachID int64, req models.AssignWeeklyTemplateRequest) (models.AssignWeeklyTemplateResponse, error) {
			return models.AssignWeeklyTemplateResponse{}, errors.New("db error")
		},
	}
	h := NewWeeklyTemplateHandler(mock)

	body, _ := json.Marshal(models.AssignWeeklyTemplateRequest{StudentID: 99, StartDate: "2024-03-04"})
	w := httptest.NewRecorder()

	h.Assign(w, newWTReq(http.MethodPost, "/api/coach/weekly-templates/3/assign", body, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
