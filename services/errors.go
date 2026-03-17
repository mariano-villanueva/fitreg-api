package services

import "errors"

// ErrNotCoach is returned when a non-coach user attempts a coach-only operation.
var ErrNotCoach = errors.New("user is not a coach")

// ErrForbidden is returned when a user attempts an operation they are not allowed to perform.
var ErrForbidden = errors.New("forbidden")

var ErrNotFound = errors.New("not found")
var ErrInvitationNotPending = errors.New("invitation is no longer pending")
var ErrStudentMaxCoaches = errors.New("student has reached the maximum number of coaches")
