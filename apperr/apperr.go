// Package apperr defines the structured error type used across the application.
package apperr

import "fmt"

// AppError is a structured error that carries an HTTP status code, the name of
// the operation that failed, a user-facing message, and the underlying cause.
//
// The Message field is written to the HTTP response body.
// The Err field is logged but never sent to the client.
type AppError struct {
	Code    int    // HTTP status code
	Op      string // operation that failed, e.g. "CoachHandler.ListStudents"
	Message string // user-facing message returned in the JSON response
	Err     error  // underlying cause — logged, never exposed to clients
}

// New creates an AppError.
func New(code int, op, message string, cause error) *AppError {
	return &AppError{Code: code, Op: op, Message: message, Err: cause}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Op, e.Message)
}

// Unwrap allows errors.Is / errors.As to traverse the chain.
func (e *AppError) Unwrap() error { return e.Err }
