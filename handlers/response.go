package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
)

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