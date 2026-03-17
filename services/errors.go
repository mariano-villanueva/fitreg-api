package services

import "errors"

// ErrNotCoach is returned when a non-coach user attempts a coach-only operation.
var ErrNotCoach = errors.New("user is not a coach")
