package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/middleware"
	"github.com/fitreg/api/models"
)

type CoachHandler struct {
	DB           *sql.DB
	Notification *NotificationHandler
}

func NewCoachHandler(db *sql.DB, nh *NotificationHandler) *CoachHandler {
	return &CoachHandler{DB: db, Notification: nh}
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
		logErr("fetch segments query", err)
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
			logErr("scan segment row", err)
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
		WHERE cs.coach_id = ? AND cs.status = 'active'
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

// AddStudent is deprecated - use POST /api/invitations instead
func (h *CoachHandler) AddStudent(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusGone, "Use POST /api/invitations to invite students")
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

	var coachID, studentID int64
	var status string
	err = h.DB.QueryRow("SELECT coach_id, student_id, status FROM coach_students WHERE id = ?", csID).Scan(&coachID, &studentID, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Relationship not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch relationship")
		return
	}

	var isAdmin bool
	if err := h.DB.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin); err != nil {
		logErr("check is admin for end relationship", err)
	}

	if coachID != userID && studentID != userID && !isAdmin {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	if status != "active" {
		writeError(w, http.StatusConflict, "Relationship is not active")
		return
	}

	_, err = h.DB.Exec("UPDATE coach_students SET status = 'finished', finished_at = NOW() WHERE id = ?", csID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to end relationship")
		return
	}

	var otherID int64
	if userID == coachID {
		otherID = studentID
	} else {
		otherID = coachID
	}
	var userName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", userID).Scan(&userName); err != nil {
		logErr("fetch user name for end relationship", err)
	}
	meta := map[string]interface{}{"user_id": userID, "user_name": userName}
	h.Notification.CreateNotification(otherID, "relationship_ended", "notif_relationship_ended_title", "notif_relationship_ended_body", meta, nil)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Relationship ended"})
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
	err = h.DB.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ? AND status = 'active'", userID, studentID).Scan(&exists)
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
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
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

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "pending" {
		query += " AND aw.status = 'pending'"
	} else if statusFilter == "finished" {
		query += " AND aw.status IN ('completed', 'skipped')"
	}

	query += " ORDER BY aw.due_date DESC"

	// Pagination
	limit := 0
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 1 && limit > 0 {
			offset = (p - 1) * limit
		}
	}

	// Count total before pagination
	var total int
	countQuery := "SELECT COUNT(*) FROM assigned_workouts aw WHERE aw.coach_id = ?"
	countArgs := []interface{}{userID}
	if studentIDStr != "" {
		if sid, err := strconv.ParseInt(studentIDStr, 10, 64); err == nil {
			countQuery += " AND aw.student_id = ?"
			countArgs = append(countArgs, sid)
		}
	}
	if statusFilter == "pending" {
		countQuery += " AND aw.status = 'pending'"
	} else if statusFilter == "finished" {
		countQuery += " AND aw.status IN ('completed', 'skipped')"
	}
	if err := h.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		logErr("count assigned workouts", err)
	}

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch assigned workouts")
		return
	}
	defer rows.Close()

	workouts := []models.AssignedWorkout{}
	for rows.Next() {
		var aw models.AssignedWorkout
		var description, notes, dueDate, expectedFields sql.NullString
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
			&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
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
			aw.DueDate = truncateDate(dueDate.String)
		}
		if expectedFields.Valid {
			aw.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = fetchSegments(h.DB, workouts[i].ID)
		h.populateImageURL(&workouts[i])
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
	err = h.DB.QueryRow("SELECT 1 FROM coach_students WHERE coach_id = ? AND student_id = ? AND status = 'active'", userID, req.StudentID).Scan(&exists)
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

	if len(req.ExpectedFields) == 0 {
		req.ExpectedFields = []string{"feeling"}
	}
	expectedFieldsJSON, err := json.Marshal(req.ExpectedFields)
	if err != nil {
		logErr("marshal expected fields", err)
	}

	log.Printf("Creating assigned workout: coach=%d student=%d title=%s type=%s due=%v", userID, req.StudentID, req.Title, req.Type, dueDateVal)
	result, err := h.DB.Exec(`
		INSERT INTO assigned_workouts (coach_id, student_id, title, description, type, distance_km, duration_seconds, notes, expected_fields, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.StudentID, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, expectedFieldsJSON, dueDateVal)
	if err != nil {
		log.Printf("ERROR creating assigned workout: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create assigned workout")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		logErr("get last insert id for assigned workout", err)
	}

	// Insert segments
	for i, seg := range req.Segments {
		log.Printf("Inserting segment %d: type=%s unit=%s intensity=%s work_unit=%s work_intensity=%s rest_unit=%s rest_intensity=%s",
			i, seg.SegmentType, seg.Unit, seg.Intensity, seg.WorkUnit, seg.WorkIntensity, seg.RestUnit, seg.RestIntensity)
		_, err := h.DB.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity)
		if err != nil {
			log.Printf("ERROR inserting segment %d: %v", i, err)
			writeError(w, http.StatusInternalServerError, "Failed to create workout segment")
			return
		}
	}

	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ?
	`, id).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
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
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)
	h.populateImageURL(&aw)

	// Emit notification for student
	var coachName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", userID).Scan(&coachName); err != nil {
		logErr("fetch coach name for workout notification", err)
	}
	notifMeta := map[string]interface{}{
		"workout_id":    aw.ID,
		"workout_title": req.Title,
		"coach_name":    coachName,
	}
	h.Notification.CreateNotification(req.StudentID, "workout_assigned", "notif_workout_assigned_title", "notif_workout_assigned_body", notifMeta, nil)

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
	var description, notes, dueDate, expectedFields sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ? AND aw.coach_id = ?
	`, awID, userID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
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
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)
	h.populateImageURL(&aw)

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

	if len(req.ExpectedFields) == 0 {
		req.ExpectedFields = []string{"feeling"}
	}
	efJSON, err := json.Marshal(req.ExpectedFields)
	if err != nil {
		logErr("marshal expected fields for update", err)
	}

	_, err = h.DB.Exec(`
		UPDATE assigned_workouts
		SET title = ?, description = ?, type = ?, distance_km = ?, duration_seconds = ?, notes = ?, expected_fields = ?, due_date = ?, updated_at = NOW()
		WHERE id = ? AND coach_id = ?
	`, req.Title, req.Description, req.Type, req.DistanceKm, req.DurationSeconds, req.Notes, efJSON, dueDateVal, awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update assigned workout")
		return
	}

	// Replace segments: delete old, insert new
	if _, err := h.DB.Exec("DELETE FROM assigned_workout_segments WHERE assigned_workout_id = ?", awID); err != nil {
		logErr("delete old segments", err)
	}
	for i, seg := range req.Segments {
		if _, err := h.DB.Exec(`
			INSERT INTO assigned_workout_segments
				(assigned_workout_id, order_index, segment_type, repetitions, value, unit, intensity,
				 work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, awID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			logErr("insert updated segment", err)
		}
	}

	// Return updated workout
	var aw models.AssignedWorkout
	var description, notes, dueDate, expectedFields sql.NullString
	var studentName string
	err = h.DB.QueryRow(`
		SELECT aw.id, aw.coach_id, aw.student_id, aw.title, aw.description, aw.type,
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
			aw.created_at, aw.updated_at, u.name as student_name
		FROM assigned_workouts aw
		JOIN users u ON u.id = aw.student_id
		WHERE aw.id = ?
	`, awID).Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
		&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
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
		aw.DueDate = truncateDate(dueDate.String)
	}
	if expectedFields.Valid {
		aw.ExpectedFields = json.RawMessage(expectedFields.String)
	}
	aw.StudentName = studentName
	aw.Segments = fetchSegments(h.DB, aw.ID)
	h.populateImageURL(&aw)

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

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for delete assigned workout", err)
	}
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
			aw.distance_km, aw.duration_seconds, aw.notes, aw.expected_fields,
			aw.result_time_seconds, aw.result_distance_km, aw.result_heart_rate, aw.result_feeling,
			aw.image_file_id, aw.status, aw.due_date,
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
		var description, notes, dueDate, expectedFields sql.NullString
		if err := rows.Scan(&aw.ID, &aw.CoachID, &aw.StudentID, &aw.Title, &description, &aw.Type,
			&aw.DistanceKm, &aw.DurationSeconds, &notes, &expectedFields,
			&aw.ResultTimeSeconds, &aw.ResultDistanceKm, &aw.ResultHeartRate, &aw.ResultFeeling,
			&aw.ImageFileID, &aw.Status, &dueDate,
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
			aw.DueDate = truncateDate(dueDate.String)
		}
		if expectedFields.Valid {
			aw.ExpectedFields = json.RawMessage(expectedFields.String)
		}
		workouts = append(workouts, aw)
	}

	for i := range workouts {
		workouts[i].Segments = fetchSegments(h.DB, workouts[i].ID)
		h.populateImageURL(&workouts[i])
	}

	writeJSON(w, http.StatusOK, workouts)
}

func (h *CoachHandler) populateImageURL(aw *models.AssignedWorkout) {
	if aw.ImageFileID == nil {
		return
	}
	var uuid string
	if err := h.DB.QueryRow("SELECT uuid FROM files WHERE id = ?", *aw.ImageFileID).Scan(&uuid); err != nil {
		return
	}
	aw.ImageURL = "/api/files/" + uuid + "/download"
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

	if req.Status == "completed" && (req.ResultFeeling == nil || *req.ResultFeeling < 1 || *req.ResultFeeling > 10) {
		writeError(w, http.StatusBadRequest, "Feeling (1-10) is required when completing a workout")
		return
	}

	result, err := h.DB.Exec(`
		UPDATE assigned_workouts SET status = ?,
			result_time_seconds = ?, result_distance_km = ?, result_heart_rate = ?, result_feeling = ?,
			image_file_id = ?, updated_at = NOW() WHERE id = ? AND student_id = ?
	`, req.Status, req.ResultTimeSeconds, req.ResultDistanceKm, req.ResultHeartRate, req.ResultFeeling, req.ImageFileID, awID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logErr("get rows affected for update status", err)
	}
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Assigned workout not found")
		return
	}

	// If completed, create a workout record for the athlete
	if req.Status == "completed" {
		var aw struct {
			Type            string
			DistanceKm      float64
			DurationSeconds int
			Notes           sql.NullString
			DueDate         sql.NullString
		}
		if err := h.DB.QueryRow(`SELECT COALESCE(type,'easy'), COALESCE(distance_km,0), COALESCE(duration_seconds,0), notes, due_date
			FROM assigned_workouts WHERE id = ?`, awID).Scan(&aw.Type, &aw.DistanceKm, &aw.DurationSeconds, &aw.Notes, &aw.DueDate); err != nil {
			logErr("fetch assigned workout for completion", err)
		}

		// Use result values if provided, fall back to assigned values
		finalDistance := aw.DistanceKm
		if req.ResultDistanceKm != nil {
			finalDistance = *req.ResultDistanceKm
		}
		finalDuration := aw.DurationSeconds
		if req.ResultTimeSeconds != nil {
			finalDuration = *req.ResultTimeSeconds
		}
		var avgHR int
		if req.ResultHeartRate != nil {
			avgHR = *req.ResultHeartRate
		}

		workoutDate := "CURDATE()"
		dateArg := interface{}(nil)
		if aw.DueDate.Valid {
			workoutDate = "?"
			dateArg = truncateDate(aw.DueDate.String)
		}

		var notes interface{}
		if aw.Notes.Valid {
			notes = aw.Notes.String
		}

		if _, err := h.DB.Exec(`INSERT INTO workouts (user_id, assigned_workout_id, date, distance_km, duration_seconds, avg_heart_rate, type, notes, created_at, updated_at)
			VALUES (?, ?, `+workoutDate+`, ?, ?, ?, ?, ?, NOW(), NOW())`,
			userID, awID, dateArg, finalDistance, finalDuration, avgHR, aw.Type, notes); err != nil {
			logErr("insert workout from completed assignment", err)
		}
	}

	// Emit notification for coach
	var studentName string
	if err := h.DB.QueryRow("SELECT COALESCE(name, '') FROM users WHERE id = ?", userID).Scan(&studentName); err != nil {
		logErr("fetch student name for status notification", err)
	}
	var coachID int64
	var workoutTitle string
	if err := h.DB.QueryRow("SELECT coach_id, title FROM assigned_workouts WHERE id = ?", awID).Scan(&coachID, &workoutTitle); err != nil {
		logErr("fetch coach id and title for status notification", err)
	}

	if req.Status == "completed" || req.Status == "skipped" {
		notifType := "workout_completed"
		notifTitle := "notif_workout_completed_title"
		notifBody := "notif_workout_completed_body"
		if req.Status == "skipped" {
			notifType = "workout_skipped"
			notifTitle = "notif_workout_skipped_title"
			notifBody = "notif_workout_skipped_body"
		}
		notifMeta := map[string]interface{}{
			"workout_id":    awID,
			"workout_title": workoutTitle,
			"student_name":  studentName,
		}
		h.Notification.CreateNotification(coachID, notifType, notifTitle, notifBody, notifMeta, nil)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Status updated", "status": req.Status})
}
