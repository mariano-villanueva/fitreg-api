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

type CoachHandler struct {
	DB *sql.DB
}

func NewCoachHandler(db *sql.DB) *CoachHandler {
	return &CoachHandler{DB: db}
}

func fetchSegments(db *sql.DB, assignedWorkoutID int64) []models.WorkoutSegment {
	rows, err := db.Query(`
		SELECT id, assigned_workout_id, order_index, segment_type, repetitions,
			value, unit, intensity, work_value, work_unit, work_intensity,
			rest_value, rest_unit, rest_intensity
		FROM assigned_workout_segments
		WHERE assigned_workout_id = ?
		ORDER BY order_index ASC
	`, assignedWorkoutID)
	if err != nil {
		return []models.WorkoutSegment{}
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.AssignedWorkoutID, &s.OrderIndex, &s.SegmentType,
			&s.Repetitions, &s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			continue
		}
		segments = append(segments, s)
	}
	return segments
}

// ListStudents handles GET /api/coach/students
func (h *CoachHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Verify user is a coach
	var isCoach bool
	err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify coach status")
		return
	}
	if !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}

	rows, err := h.DB.Query(`
		SELECT u.id, u.name, u.email, COALESCE(u.avatar_url, '') as avatar_url
		FROM users u
		JOIN coach_students cs ON u.id = cs.student_id
		WHERE cs.coach_id = ?
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch students")
		return
	}
	defer rows.Close()

	type StudentInfo struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	students := []StudentInfo{}
	for rows.Next() {
		var s StudentInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.AvatarURL); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to scan student")
			return
		}
		students = append(students, s)
	}

	writeJSON(w, http.StatusOK, students)
}

// AddStudent handles POST /api/coach/students
func (h *CoachHandler) AddStudent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Verify user is a coach
	var isCoach bool
	err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify coach status")
		return
	}
	if !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
		return
	}

	var req models.AddStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Find student by email
	var studentID int64
	var studentName, studentEmail, studentAvatar string
	var avatar sql.NullString
	err = h.DB.QueryRow("SELECT id, name, email, avatar_url FROM users WHERE email = ?", req.Email).Scan(
		&studentID, &studentName, &studentEmail, &avatar,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "User not found with that email")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to find user")
		return
	}

	if avatar.Valid {
		studentAvatar = avatar.String
	}

	// Cannot add self
	if studentID == userID {
		writeError(w, http.StatusBadRequest, "Cannot add yourself as a student")
		return
	}

	// Insert relationship
	_, err = h.DB.Exec("INSERT INTO coach_students (coach_id, student_id) VALUES (?, ?)", userID, studentID)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			writeError(w, http.StatusConflict, "Student already added")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to add student")
		return
	}

	type StudentInfo struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	writeJSON(w, http.StatusCreated, StudentInfo{
		ID:        studentID,
		Name:      studentName,
		Email:     studentEmail,
		AvatarURL: studentAvatar,
	})
}

// RemoveStudent handles DELETE /api/coach/students/{id}
func (h *CoachHandler) RemoveStudent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	studentID, err := extractID(r.URL.Path, "/api/coach/students/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	result, err := h.DB.Exec("DELETE FROM coach_students WHERE coach_id = ? AND student_id = ?", userID, studentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove student")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Student relationship not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Student removed"})
}

// GetStudentWorkouts handles GET /api/coach/students/{id}/workouts
func (h *CoachHandler) GetStudentWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Extract student ID from path (before /workouts)
	path := strings.TrimSuffix(r.URL.Path, "/workouts")
	studentID, err := extractID(path, "/api/coach/students/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	// Verify student belongs to this coach
	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ?", userID, studentID).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusForbidden, "Student does not belong to this coach")
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, user_id, date, distance_km, duration_seconds, avg_pace, calories, avg_heart_rate, type, notes, created_at, updated_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY date DESC
	`, studentID)
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

// ListAssignedWorkouts handles GET /api/coach/assigned-workouts
func (h *CoachHandler) ListAssignedWorkouts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	query := `
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.coach_id = ?
	`
	args := []interface{}{userID}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr != "" {
		sid, err := strconv.ParseInt(studentIDStr, 10, 64)
		if err == nil {
			query += " AND aw.student_id = ?"
			args = append(args, sid)
		}
	}

	query += " ORDER BY aw.due_date DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workouts")
		return
	}
	defer rows.Close()

	workouts := []models.AssignedWorkout{}
	for rows.Next() {
		var aw models.AssignedWorkout
		var description, notes, dueDate sql.NullString
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
			&aw.DistanceKm, &aw.DurationSeconds, &notes, &aw.Status, &dueDate,
			&aw.CreatedAt, &aw.UpdatedAt, &aw.StudentName); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to scan assigned workout")
			return
		}
		if description.Valid {
			aw.Description = description.String
		}
		if notes.Valid {
			aw.Notes = notes.String
		}
		if dueDate.Valid {
			aw.DueDate = dueDate.String
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = fetchSegments(h.DB, workouts[i].ID)
	}

	writeJSON(w, http.StatusOK, workouts)
}

// CreateAssignedWorkout handles POST /api/coach/assigned-workouts
func (h *CoachHandler) CreateAssignedWorkout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Verify user is a coach
	var isCoach bool
	err := h.DB.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", userID).Scan(&isCoach)
	if err != nil || !isCoach {
		writeError(w, http.StatusForbidden, "User is not a coach")
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

	// Verify student belongs to coach
	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ?", userID, req.StudentID).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusForbidden, "Student does not belong to this coach")
		return
	}

	var dueDateVal interface{}
	if req.DueDate != "" {
		dueDateVal = req.DueDate
	} else {
		dueDateVal = nil
	}

	result, err := h.DB.Exec(`
		INSERT INTO assigned_workouts (coach_id, student_id, title, description, type, distance_km, duration_seconds, notes, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.StudentID, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, dueDateVal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create assigned workout")
		return
	}

	id, _ := result.LastInsertId()

	// Insert segments
	for i, seg := range req.Segments {
		_, err := h.DB.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create workout segment")
			return
		}
	}

	var aw models.AssignedWorkout
	var description, notes, dueDate sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ?
	`, id).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt, &studentName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch created workout")
		return
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = dueDate.String
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)

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

	var aw models.AssignedWorkout
	var description, notes, dueDate sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ? AND aw.coach_id = ?
	`, awID, userID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt, &studentName)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workout")
		return
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = dueDate.String
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)

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

	// Check workout exists, belongs to coach, and is not completed
	var status string
	err = h.DB.QueryRow("SELECT status FROM assigned_workouts WHERE id = ? AND coach_id = ?", awID, userID).Scan(&status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch workout")
		return
	}
	if status == "completed" {
		writeError(w, http.StatusBadRequest, "Cannot edit a completed workout")
		return
	}

	var req models.UpdateAssignedWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var dueDateVal interface{}
	if req.DueDate != "" {
		dueDateVal = req.DueDate
	} else {
		dueDateVal = nil
	}

	_, err = h.DB.Exec(`
		UPDATE assigned_workouts
		SET title = ?, description = ?, type = ?, distance_km = ?, duration_seconds = ?, notes = ?, due_date = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, dueDateVal, awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update assigned workout")
		return
	}

	// Replace segments: delete old, insert new
	h.DB.Exec("DELETE FROM assigned_workout_segments WHERE assigned_workout_id = ?", awID)
	for i, seg := range req.Segments {
		h.DB.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, awID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity)
	}

	// Return updated workout
	var aw models.AssignedWorkout
	var description, notes, dueDate sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ?
	`, awID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &aw.Status, &dueDate,
		&aw.CreatedAt, &aw.UpdatedAt, &studentName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch updated workout")
		return
	}
	if description.Valid {
		aw.Description = description.String
	}
	if notes.Valid {
		aw.Notes = notes.String
	}
	if dueDate.Valid {
		aw.DueDate = dueDate.String
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)

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

	result, err := h.DB.Exec("DELETE FROM assigned_workouts WHERE id = ? AND coach_id = ?", awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete assigned workout")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
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

	rows, err := h.DB.Query(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as coach_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.coach_id
		WHERE aw.student_id = ?
		ORDER BY aw.due_date ASC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workouts")
		return
	}
	defer rows.Close()

	workouts := []models.AssignedWorkout{}
	for rows.Next() {
		var aw models.AssignedWorkout
		var description, notes, dueDate sql.NullString
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
			&aw.DistanceKm, &aw.DurationSeconds, &notes, &aw.Status, &dueDate,
			&aw.CreatedAt, &aw.UpdatedAt, &aw.CoachName); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to scan assigned workout")
			return
		}
		if description.Valid {
			aw.Description = description.String
		}
		if notes.Valid {
			aw.Notes = notes.String
		}
		if dueDate.Valid {
			aw.DueDate = dueDate.String
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = fetchSegments(h.DB, workouts[i].ID)
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

	// Extract ID from path: /api/my-assigned-workouts/{id}/status
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

	result, err := h.DB.Exec("UPDATE assigned_workouts SET status = ?, updated_at = NOW() WHERE id = ? AND student_id = ?", req.Status, awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Status updated", "status": req.Status})
}
