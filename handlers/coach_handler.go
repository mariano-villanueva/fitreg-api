package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
)

type CoachHandler struct {
	svc CoachServicer
}

func NewCoachHandler(svc CoachServicer) *CoachHandler {
	return &CoachHandler{svc: svc}
}

// ListStudents handles GET /api/coach/students
func (h *CoachHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	students, err := h.svc.ListStudents(userID)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.ListStudents", apperr.COACH_001, "Failed to fetch students")
		return
	}

	writeJSON(w, http.StatusOK, students)
}

// EndRelationship handles PUT /api/coach-students/{id}/end
func (h *CoachHandler) EndRelationship(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/end")
	csID, err := extractID(path, "/api/coach-students/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid relationship ID")
		return
	}

	if err := h.svc.EndRelationship(csID, userID); err != nil {
		handleServiceErr(w, err, "CoachHandler.EndRelationship", apperr.COACH_002, "Failed to end relationship")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Relationship ended"})
}

// GetStudentWorkouts handles GET /api/coach/students/{id}/workouts
func (h *CoachHandler) GetStudentWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/workouts")
	studentID, err := extractID(path, "/api/coach/students/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	workouts, err := h.svc.GetStudentWorkouts(userID, studentID)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetStudentWorkouts", apperr.COACH_003, "Failed to fetch workouts")
		return
	}

	writeJSON(w, http.StatusOK, workouts)
}

// GetDailySummary handles GET /api/coach/daily-summary?date=YYYY-MM-DD
func (h *CoachHandler) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	includeSegments := r.URL.Query().Get("segments") != "false"
	items, err := h.svc.GetDailySummary(userID, date, includeSegments)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetDailySummary", apperr.COACH_011, "Failed to fetch daily summary")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// GetStudentLoad handles GET /api/coach/students/{id}/load?weeks=N
func (h *CoachHandler) GetStudentLoad(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract student ID from path: /api/coach/students/{id}/load
	path := strings.TrimPrefix(r.URL.Path, "/api/coach/students/")
	path = strings.TrimSuffix(path, "/load")
	studentID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || studentID == 0 {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	weeks := parseWeeksParam(r, 8)

	load, err := h.svc.GetStudentLoad(coachID, studentID, weeks)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetStudentLoad", apperr.COACH_012, "Failed to fetch student load")
		return
	}
	writeJSON(w, http.StatusOK, load)
}

// GetMyLoad handles GET /api/me/load?weeks=N
func (h *CoachHandler) GetMyLoad(w http.ResponseWriter, r *http.Request) {
	studentID := middleware.UserIDFromContext(r.Context())
	if studentID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	weeks := parseWeeksParam(r, 8)

	load, err := h.svc.GetMyLoad(studentID, weeks)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetMyLoad", apperr.COACH_013, "Failed to fetch training load")
		return
	}
	writeJSON(w, http.StatusOK, load)
}

// parseWeeksParam reads the ?weeks= query param, defaulting to defaultVal and capping at 52.
func parseWeeksParam(r *http.Request, defaultVal int) int {
	weeks := defaultVal
	if w := r.URL.Query().Get("weeks"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			weeks = n
		}
	}
	if weeks > 52 {
		weeks = 52
	}
	return weeks
}
