package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/fitreg/api/apperr"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeAppError logs the full error context and writes the user-facing message
// as a JSON response. Always logs regardless of status code; the underlying
// cause (ae.Err) is included in the log but never sent to the client.
func writeAppError(w http.ResponseWriter, ae *apperr.AppError) {
	if ae.Err != nil {
		log.Printf("ERROR [%s] HTTP %d: %s — %v", ae.Op, ae.Code, ae.Message, ae.Err)
	} else {
		log.Printf("ERROR [%s] HTTP %d: %s", ae.Op, ae.Code, ae.Message)
	}
	writeJSON(w, ae.Code, map[string]string{"error": ae.Message})
}

// writeError writes a simple error response and logs it.
// Use writeAppError when you have an underlying error to preserve.
func writeError(w http.ResponseWriter, status int, message string) {
	log.Printf("ERROR HTTP %d: %s", status, message)
	writeJSON(w, status, map[string]string{"error": message})
}

// logErr logs a non-fatal error with its context string.
func logErr(context string, err error) {
	if err == nil {
		return
	}
	log.Printf("WARN [%s]: %v", context, err)
}
