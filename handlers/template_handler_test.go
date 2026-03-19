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

type mockTemplateService struct {
	createFn func(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error)
	listFn   func(coachID int64) ([]models.WorkoutTemplate, error)
	getFn    func(id, coachID int64) (models.WorkoutTemplate, error)
	updateFn func(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error)
	deleteFn func(id, coachID int64) error
}

func (m *mockTemplateService) Create(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	return m.createFn(coachID, req)
}
func (m *mockTemplateService) List(coachID int64) ([]models.WorkoutTemplate, error) {
	return m.listFn(coachID)
}
func (m *mockTemplateService) Get(id, coachID int64) (models.WorkoutTemplate, error) {
	return m.getFn(id, coachID)
}
func (m *mockTemplateService) Update(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
	return m.updateFn(id, coachID, req)
}
func (m *mockTemplateService) Delete(id, coachID int64) error {
	return m.deleteFn(id, coachID)
}

func newTmplReq(method, path string, body []byte, userID int64) *http.Request {
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

func TestTemplateHandler_List_ReturnsTemplates(t *testing.T) {
	mock := &mockTemplateService{
		listFn: func(coachID int64) ([]models.WorkoutTemplate, error) {
			return []models.WorkoutTemplate{{ID: 1, Title: "5k Easy"}}, nil
		},
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.List(w, newTmplReq(http.MethodGet, "/api/coach/templates", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []models.WorkoutTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != 1 {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestTemplateHandler_List_ServiceError_Returns500(t *testing.T) {
	mock := &mockTemplateService{
		listFn: func(coachID int64) ([]models.WorkoutTemplate, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.List(w, newTmplReq(http.MethodGet, "/api/coach/templates", nil, 10))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Create ---

func TestTemplateHandler_Create_ReturnsCreated(t *testing.T) {
	mock := &mockTemplateService{
		createFn: func(coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
			return models.WorkoutTemplate{ID: 5, Title: req.Title}, nil
		},
	}
	h := NewTemplateHandler(mock)

	body, _ := json.Marshal(models.CreateTemplateRequest{
		Title:    "Interval Training",
		Segments: []models.SegmentRequest{{SegmentType: "interval"}},
	})
	w := httptest.NewRecorder()

	h.Create(w, newTmplReq(http.MethodPost, "/api/coach/templates", body, 10))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp models.WorkoutTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 5 || resp.Title != "Interval Training" {
		t.Errorf("unexpected body: %+v", resp)
	}
}

func TestTemplateHandler_Create_MissingTitle_Returns400(t *testing.T) {
	h := NewTemplateHandler(&mockTemplateService{})

	body, _ := json.Marshal(models.CreateTemplateRequest{
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.Create(w, newTmplReq(http.MethodPost, "/api/coach/templates", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTemplateHandler_Create_NoSegments_Returns400(t *testing.T) {
	h := NewTemplateHandler(&mockTemplateService{})

	body, _ := json.Marshal(models.CreateTemplateRequest{Title: "Run"})
	w := httptest.NewRecorder()

	h.Create(w, newTmplReq(http.MethodPost, "/api/coach/templates", body, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Get ---

func TestTemplateHandler_Get_ReturnsTemplate(t *testing.T) {
	mock := &mockTemplateService{
		getFn: func(id, coachID int64) (models.WorkoutTemplate, error) {
			return models.WorkoutTemplate{ID: id, Title: "5k"}, nil
		},
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Get(w, newTmplReq(http.MethodGet, "/api/coach/templates/3", nil, 10))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp models.WorkoutTemplate
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 3 {
		t.Errorf("expected ID 3, got %d", resp.ID)
	}
}

func TestTemplateHandler_Get_InvalidID_Returns400(t *testing.T) {
	h := NewTemplateHandler(&mockTemplateService{})
	w := httptest.NewRecorder()

	h.Get(w, newTmplReq(http.MethodGet, "/api/coach/templates/abc", nil, 10))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTemplateHandler_Get_NotFound_Returns404(t *testing.T) {
	mock := &mockTemplateService{
		getFn: func(id, coachID int64) (models.WorkoutTemplate, error) {
			return models.WorkoutTemplate{}, services.ErrNotFound
		},
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Get(w, newTmplReq(http.MethodGet, "/api/coach/templates/99", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Update ---

func TestTemplateHandler_Update_Returns200(t *testing.T) {
	mock := &mockTemplateService{
		updateFn: func(id, coachID int64, req models.CreateTemplateRequest) (models.WorkoutTemplate, error) {
			return models.WorkoutTemplate{ID: id, Title: req.Title}, nil
		},
	}
	h := NewTemplateHandler(mock)

	body, _ := json.Marshal(models.CreateTemplateRequest{
		Title:    "Updated",
		Segments: []models.SegmentRequest{{SegmentType: "simple"}},
	})
	w := httptest.NewRecorder()

	h.Update(w, newTmplReq(http.MethodPut, "/api/coach/templates/3", body, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Delete ---

func TestTemplateHandler_Delete_Returns200(t *testing.T) {
	mock := &mockTemplateService{
		deleteFn: func(id, coachID int64) error { return nil },
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Delete(w, newTmplReq(http.MethodDelete, "/api/coach/templates/3", nil, 10))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestTemplateHandler_Delete_NotFound_Returns404(t *testing.T) {
	mock := &mockTemplateService{
		deleteFn: func(id, coachID int64) error { return services.ErrNotFound },
	}
	h := NewTemplateHandler(mock)
	w := httptest.NewRecorder()

	h.Delete(w, newTmplReq(http.MethodDelete, "/api/coach/templates/99", nil, 10))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
