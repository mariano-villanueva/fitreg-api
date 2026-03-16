package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
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
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
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
		if err := rows.Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
			&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
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
		wo.Segments = h.fetchWorkoutSegments(wo.ID)
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

	// Verify workout exists and belongs to user
	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM workouts WHERE id = ? AND user_id = ?", id, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workout")
		return
	}

	wo := h.fetchWorkout(id)
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

	result, err := h.DB.Exec(`
		INSERT INTO workouts (user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create workout")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for workout", err)
	}

	for i, seg := range req.Segments {
		if _, err := h.DB.Exec(`
			INSERT INTO workout_segments (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create workout segment")
			return
		}
	}

	wo := h.fetchWorkout(id)
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

	result, err := h.DB.Exec(`
		UPDATE workouts SET date = ?, distance_km = ?, duration_seconds = ?, avg_pace = ?, calories = ?, avg_heart_rate = ?, feeling = ?, type = ?, notes = ?, updated_at = NOW()
		WHERE id = ? AND user_id = ?
	`, req.Date, req.DistanceKm, req.DurationSeconds, req.AvgPace, req.Calories, req.AvgHeartRate, req.Feeling, req.Type, req.Notes, id, userID)
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

	// Replace segments
	if _, err := h.DB.Exec("DELETE FROM workout_segments WHERE workout_id = ?", id); err != nil {
		logErr("delete old workout segments", err)
	}
	for i, seg := range req.Segments {
		if _, err := h.DB.Exec(`
			INSERT INTO workout_segments (workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			logErr("insert updated workout segment", err)
		}
	}

	wo := h.fetchWorkout(id)
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

func (h *WorkoutHandler) fetchWorkout(id int64) models.Workout {
	var wo models.Workout
	var avgPace, workoutType, notes sql.NullString
	if err := h.DB.QueryRow(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, feeling, type, notes, created_at, updated_at
		FROM workouts WHERE id = ?
	`, id).Scan(&wo.ID, &wo.UserID, &wo.Date, &wo.DistanceKm, &wo.DurationSeconds,
		&avgPace, &wo.Calories, &wo.AvgHeartRate, &wo.Feeling, &workoutType, &notes, &wo.CreatedAt, &wo.UpdatedAt); err != nil {
		logErr("fetch workout", err)
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
	wo.Segments = h.fetchWorkoutSegments(id)
	return wo
}

func (h *WorkoutHandler) fetchWorkoutSegments(workoutID int64) []models.WorkoutSegment {
	rows, err := h.DB.Query(`
		SELECT id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
			COALESCE(value, 0), COALESCE(unit, ''), COALESCE(intensity, ''),
			COALESCE(work_value, 0), COALESCE(work_unit, ''), COALESCE(work_intensity, ''),
			COALESCE(rest_value, 0), COALESCE(rest_unit, ''), COALESCE(rest_intensity, '')
		FROM workout_segments WHERE workout_id = ? ORDER BY order_index
	`, workoutID)
	if err != nil {
		logErr("fetch workout segments", err)
		return []models.WorkoutSegment{}
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			logErr("scan workout segment", err)
			continue
		}
		segments = append(segments, s)
	}
	return segments
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

