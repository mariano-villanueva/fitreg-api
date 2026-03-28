package services

import "errors"

// ErrNotCoach is returned when a non-coach user attempts a coach-only operation.
var ErrNotCoach = errors.New("user is not a coach")

// ErrForbidden is returned when a user attempts an operation they are not allowed to perform.
var ErrForbidden = errors.New("forbidden")

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvitationNotPending is returned when acting on a non-pending invitation.
var ErrInvitationNotPending = errors.New("invitation is no longer pending")

// ErrStudentMaxCoaches is returned when a student already has the maximum number of coaches.
var ErrStudentMaxCoaches = errors.New("student has reached the maximum number of coaches")

// ErrWorkoutFinished is returned when trying to edit or delete a non-pending assigned workout.
var ErrWorkoutFinished = errors.New("cannot edit a finished workout")

// ErrInvalidToken is returned when an invite token is not found or already redeemed.
var ErrInvalidToken = errors.New("invalid_token")
