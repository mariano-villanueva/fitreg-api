package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type WorkoutHandler struct {
	DB *sql.DB
}

func NewWorkoutHandler(db *sql.DB) *WorkoutHandler {
	return &WorkoutHandler{DB: db}
}

func (h *WorkoutHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, user_id, assigned_workout_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY date DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workouts")
		return
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var wo models.Workout
		var avgPace, workoutType, notes sql.NullString
		if err := rows.Scan(&wo.ID, &wo.UserID, &wo.AssignedWorkoutID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
			&avgPace, &wo.Calories, &wo.AvgHeartRate, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to scan workout")
			return
		}
		if avgPace.Valid {
			wo.AvgPace = avgPace.String
		}
		if workoutType.Valid {
			wo.Type = workoutType.String
		}
		if notes.Valid {
			wo.Notes = notes.String
		}
		workouts = append(workouts, wo)
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

	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	err = h.DB.QueryRow(`
		SELECT id, user_id, assigned_workout_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes, created_at, updated_at
		FROM workouts WHERE id = ? AND user_id = ?
	`, id, userID).Scan(&wo.ID, &wo.UserID, &wo.AssignedWorkoutID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workout")
		return
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
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

	result, err := h.DB.Exec(`
		INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Type, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create workout")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for workout", err)
	}

	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	if err := h.DB.QueryRow(`
		SELECT id, user_id, assigned_workout_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes, created_at, updated_at
		FROM workouts WHERE id = ?
	`, id).Scan(&wo.ID, &wo.UserID, &wo.AssignedWorkoutID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
		logErr("fetch created workout", err)
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
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

	result, err := h.DB.Exec(`
		UPDATE workouts SET date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?, avg_heart_rate = ?, type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ?
	`, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Type, req.Notes, id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update workout")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for update workout", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}

	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	if err := h.DB.QueryRow(`
		SELECT id, user_id, assigned_workout_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes, created_at, updated_at
		FROM workouts WHERE id = ?
	`, id).Scan(&wo.ID, &wo.UserID, &wo.AssignedWorkoutID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
		logErr("fetch updated workout", err)
	}
	if avgPace.Valid {
		wo.AvgPace = avgPace.String
	}
	if workoutType.Valid {
		wo.Type = workoutType.String
	}
	if notes.Valid {
		wo.Notes = notes.String
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

	result, err := h.DB.Exec(`DELETE FROM workouts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete workout")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for delete workout", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Workout deleted"})
}

// extractID parses the numeric ID from a URL path given a prefix.
func extractID(path, prefix string) (int64, error) {
	s := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	return strconv.ParseInt(s, 10, 64)
}

func truncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if status >= 500 {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("ERROR [%s:%d] %d: %s", file, line, status, message)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

// logErr logs an error with caller context. Use for errors that are handled
// but should be visible in logs for debugging.
func logErr(context string, err error) {
	if err == nil {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	log.Printf("ERROR [%s:%d] %s: %v", file, line, context, err)
}
