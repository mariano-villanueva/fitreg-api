package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
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

// ListAssignedWorkouts handles GET /api/coach/assigned-workouts
func (h *CoachHandler) ListAssignedWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var studentID int64
	if studentIDStr := r.URL.Query().Get("student_id"); studentIDStr != "" {
		if sid, err := strconv.ParseInt(studentIDStr, 10, 64); err == nil {
			studentID = sid
		}
	}

	statusFilter := r.URL.Query().Get("status")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	const maxPageLimit = 100
	limit := 0
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > maxPageLimit {
				l = maxPageLimit
			}
			limit = l
		}
	}
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 1 && limit > 0 {
			offset = (p - 1) * limit
		}
	}

	workouts, total, err := h.svc.ListAssignedWorkouts(userID, studentID, statusFilter, startDate, endDate, limit, offset)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.ListAssignedWorkouts", apperr.COACH_004, "Failed to fetch assigned workouts")
		return
	}

	if limit > 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":  workouts,
			"total": total,
		})
	} else {
		writeJSON(w, http.StatusOK, workouts)
	}
}

// CreateAssignedWorkout handles POST /api/coach/assigned-workouts
func (h *CoachHandler) CreateAssignedWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateAssignedWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	aw, err := h.svc.CreateAssignedWorkout(userID, req)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.CreateAssignedWorkout", apperr.COACH_005, "Failed to create assigned workout")
		return
	}

	writeJSON(w, http.StatusCreated, aw)
}

// GetAssignedWorkout handles GET /api/coach/assigned-workouts/{id}
func (h *CoachHandler) GetAssignedWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/coach/assigned-workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	aw, err := h.svc.GetAssignedWorkout(awID, userID)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetAssignedWorkout", apperr.COACH_006, "Failed to fetch assigned workout")
		return
	}

	writeJSON(w, http.StatusOK, aw)
}

// UpdateAssignedWorkout handles PUT /api/coach/assigned-workouts/{id}
func (h *CoachHandler) UpdateAssignedWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/coach/assigned-workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	var req models.UpdateAssignedWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}

	aw, err := h.svc.UpdateAssignedWorkout(awID, userID, req)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.UpdateAssignedWorkout", apperr.COACH_007, "Failed to update assigned workout")
		return
	}

	writeJSON(w, http.StatusOK, aw)
}

// DeleteAssignedWorkout handles DELETE /api/coach/assigned-workouts/{id}
func (h *CoachHandler) DeleteAssignedWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	awID, err := extractID(r.URL.Path, "/api/coach/assigned-workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	if err := h.svc.DeleteAssignedWorkout(awID, userID); err != nil {
		handleServiceErr(w, err, "CoachHandler.DeleteAssignedWorkout", apperr.COACH_008, "Failed to delete assigned workout")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Assigned workout deleted"})
}

// GetMyAssignedWorkouts handles GET /api/my-assigned-workouts
func (h *CoachHandler) GetMyAssignedWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	workouts, err := h.svc.GetMyAssignedWorkouts(userID, startDate, endDate)
	if err != nil {
		handleServiceErr(w, err, "CoachHandler.GetMyAssignedWorkouts", apperr.COACH_009, "Failed to fetch assigned workouts")
		return
	}

	writeJSON(w, http.StatusOK, workouts)
}

// UpdateAssignedWorkoutStatus handles PUT /api/my-assigned-workouts/{id}/status
func (h *CoachHandler) UpdateAssignedWorkoutStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/status")
	awID, err := extractID(path, "/api/my-assigned-workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assigned workout ID")
		return
	}

	var req models.UpdateAssignedWorkoutStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status != "pending" && req.Status != "completed" && req.Status != "skipped" {
		writeError(w, http.StatusBadRequest, "Invalid status. Must be pending, completed, or skipped")
		return
	}

	if req.Status == "completed" && (req.ResultFeeling == nil || *req.ResultFeeling < 1 || *req.ResultFeeling > 10) {
		writeError(w, http.StatusBadRequest, "Feeling (1-10) is required when completing a workout")
		return
	}

	// Sanity bounds on optional numeric result fields to prevent garbage data.
	if req.ResultDistanceKm != nil && (*req.ResultDistanceKm < 0 || *req.ResultDistanceKm > 1000) {
		writeError(w, http.StatusBadRequest, "result_distance_km must be between 0 and 1000")
		return
	}
	if req.ResultTimeSeconds != nil && (*req.ResultTimeSeconds < 0 || *req.ResultTimeSeconds > 86400*7) {
		writeError(w, http.StatusBadRequest, "result_time_seconds must be between 0 and 604800")
		return
	}
	if req.ResultHeartRate != nil && (*req.ResultHeartRate < 0 || *req.ResultHeartRate > 300) {
		writeError(w, http.StatusBadRequest, "result_heart_rate must be between 0 and 300")
		return
	}

	if err := h.svc.UpdateAssignedWorkoutStatus(awID, userID, req); err != nil {
		handleServiceErr(w, err, "CoachHandler.UpdateAssignedWorkoutStatus", apperr.COACH_010, "Failed to update status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Status updated", "status": req.Status})
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
