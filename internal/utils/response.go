package utils

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse defines the standard error response JSON format.
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON writes a JSON response with status code and Content-Type header.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{Error: message})
}

