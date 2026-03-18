package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type WorkoutHandler struct {
	svc *services.WorkoutService
}

func NewWorkoutHandler(svc *services.WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{svc: svc}
}

func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	workouts, err := h.svc.List(userID)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.ListWorkouts", "Failed to fetch workouts")
		return
	}
	writeJSON(w, http.StatusOK, workouts)
}

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
		handleServiceErr(w, err, "WorkoutHandler.GetWorkout", "Failed to fetch workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

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
	if req.Date == "" {
		writeError(w, http.StatusBadRequest, "date is required")
		return
	}
	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}
	wo, err := h.svc.Create(userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.CreateWorkout", "Failed to create workout")
		return
	}
	writeJSON(w, http.StatusCreated, wo)
}

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
	if len(req.Segments) == 0 {
		writeError(w, http.StatusBadRequest, "at least one segment is required")
		return
	}
	wo, err := h.svc.Update(id, userID, req)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.UpdateWorkout", "Failed to update workout")
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

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
	err = h.svc.Delete(id, userID)
	if err != nil {
		handleServiceErr(w, err, "WorkoutHandler.DeleteWorkout", "Failed to delete workout")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}
