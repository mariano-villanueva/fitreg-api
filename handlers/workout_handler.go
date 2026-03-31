package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type WorkoutHandler struct {
	svc WorkoutServicer
}

func NewWorkoutHandler(svc WorkoutServicer) *WorkoutHandler {
	return &WorkoutHandler{svc: svc}
}

// ListWorkouts handles GET /api/workouts
func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	workouts, err := h.svc.List(userID, startDate, endDate)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.ListWorkouts", apperr.WORKOUT_001, "Failed to fetch workouts")
		return
	}
	writeJSON(w, http.StatusOK, workouts)
}

// GetWorkout handles GET /api/workouts/{id}
func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	wo, err := h.svc.GetByID(id, userID)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.GetWorkout", apperr.WORKOUT_002, "Failed to fetch workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// CreateWorkout handles POST /api/workouts
func (h *WorkoutHandler) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.DueDate == "" {
		writeError(w, http.StatusBadRequest, "due_date is required")
		return
	}
	wo, err := h.svc.Create(userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.CreateWorkout", apperr.WORKOUT_003, "Failed to create workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

// UpdateWorkout handles PUT /api/workouts/{id}
func (h *WorkoutHandler) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	var req models.UpdateWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	wo, err := h.svc.Update(id, userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateWorkout", apperr.WORKOUT_004, "Failed to update workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// DeleteWorkout handles DELETE /api/workouts/{id}
func (h *WorkoutHandler) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	if err := h.svc.Delete(id, userID); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.DeleteWorkout", apperr.WORKOUT_005, "Failed to delete workout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}

// UpdateWorkoutStatus handles PUT /api/workouts/{id}/status
func (h *WorkoutHandler) UpdateWorkoutStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/status")
	id, err := extractID(path, "/api/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	var req models.UpdateWorkoutStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Status != "completed" && req.Status != "skipped" {
		writeError(w, http.StatusBadRequest, "status must be completed or skipped")
		return
	}
	if req.Status == "completed" && (req.ResultFeeling == nil || *req.ResultFeeling < 1 || *req.ResultFeeling > 10) {
		writeError(w, http.StatusBadRequest, "feeling (1-10) is required when completing a workout")
		return
	}
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
	if err := h.svc.UpdateStatus(id, userID, req); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateWorkoutStatus", apperr.WORKOUT_006, "Failed to update workout status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Status updated", "status": req.Status})
}

// ListCoachWorkouts handles GET /api/coach/workouts
func (h *WorkoutHandler) ListCoachWorkouts(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var studentID *int64
	if s := r.URL.Query().Get("student_id"); s != "" {
		if sid, err := strconv.ParseInt(s, 10, 64); err == nil {
			studentID = &sid
		}
	}
	statusFilter := r.URL.Query().Get("status")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	const maxPageLimit = 100
	limit, offset := 0, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > maxPageLimit {
				n = maxPageLimit
			}
			limit = n
		}
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 && limit > 0 {
			offset = (n - 1) * limit
		}
	}
	workouts, total, err := h.svc.ListCoachWorkouts(coachID, studentID, statusFilter, startDate, endDate, limit, offset)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.ListCoachWorkouts", apperr.WORKOUT_007, "Failed to fetch coach workouts")
		return
	}
	if limit > 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": workouts, "total": total})
	} else {
		writeJSON(w, http.StatusOK, workouts)
	}
}

// CreateCoachWorkout handles POST /api/coach/workouts
func (h *WorkoutHandler) CreateCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req models.CreateCoachWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	wo, err := h.svc.CreateCoachWorkout(coachID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.CreateCoachWorkout", apperr.WORKOUT_008, "Failed to create coach workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

// GetCoachWorkout handles GET /api/coach/workouts/{id}
func (h *WorkoutHandler) GetCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	wo, err := h.svc.GetCoachWorkout(id, coachID)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.GetCoachWorkout", apperr.WORKOUT_009, "Failed to fetch coach workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// UpdateCoachWorkout handles PUT /api/coach/workouts/{id}
func (h *WorkoutHandler) UpdateCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	var req models.UpdateCoachWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	wo, err := h.svc.UpdateCoachWorkout(id, coachID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateCoachWorkout", apperr.WORKOUT_010, "Failed to update coach workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

// DeleteCoachWorkout handles DELETE /api/coach/workouts/{id}
func (h *WorkoutHandler) DeleteCoachWorkout(w http.ResponseWriter, r *http.Request) {
	coachID := middleware.UserIDFromContext(r.Context())
	if coachID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := extractID(r.URL.Path, "/api/coach/workouts/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid workout ID")
		return
	}
	if err := h.svc.DeleteCoachWorkout(id, coachID); err != nil {
		handleServiceErr(w, err, "WorkoutHandler.DeleteCoachWorkout", apperr.WORKOUT_011, "Failed to delete coach workout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}
