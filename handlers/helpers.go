package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitreg/api/apperr"
	"github.com/fitreg/api/services"
)

// extractID parses the numeric ID from a URL path given a prefix.
func extractID(path, prefix string) (int64, error) {
	s := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	return strconv.ParseInt(s, 10, 64)
}

// handleServiceErr maps well-known service and repository errors to their
// appropriate HTTP status codes and writes the response. Unknown errors fall
// through to 500 with the provided fallbackMsg.
//
// Use this instead of a manual errors.Is chain in every handler:
//
//	if err != nil {
//	    handleServiceErr(w, err, "CoachHandler.ListStudents", "Failed to list students")
//	    return
//	}
func handleServiceErr(w http.ResponseWriter, err error, op, internalCode, fallbackMsg string) {
	var code int
	var msg string

	switch {
	case errors.Is(err, services.ErrNotCoach):
		code, msg = http.StatusForbidden, "User is not a coach"
	case errors.Is(err, services.ErrForbidden):
		code, msg = http.StatusForbidden, "Access denied"
	case errors.Is(err, services.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		code, msg = http.StatusNotFound, "Not found"
	case errors.Is(err, services.ErrInvitationNotPending):
		code, msg = http.StatusConflict, "Invitation is no longer pending"
	case errors.Is(err, services.ErrStudentMaxCoaches):
		code, msg = http.StatusConflict, "Student has reached the maximum number of coaches"
	case errors.Is(err, services.ErrWorkoutFinished):
		code, msg = http.StatusConflict, "Cannot edit a finished workout"
	default:
		code = http.StatusInternalServerError
		msg = fallbackMsg
	}

	writeAppError(w, apperr.New(code, op, internalCode, msg, err))
}
