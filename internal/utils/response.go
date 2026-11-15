package utils

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorResponse defines the standard error response JSON format.
type ErrorResponse struct {
	Error  string            `json:"error"`
	Errors map[string]string `json:"errors,omitempty"`
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

// ValidationError writes a JSON response for validation errors.
// If err is (or wraps) ValidationErrors, it outputs HTTP 400 with field-level details.
// Otherwise it outputs a standard HTTP 400 with err.Error().
func ValidationError(w http.ResponseWriter, err error) {
	var valErr ValidationErrors
	if errors.As(err, &valErr) {
		JSON(w, http.StatusBadRequest, ErrorResponse{
			Error:  "validation failed",
			Errors: valErr,
		})
		return
	}
	Error(w, http.StatusBadRequest, err.Error())
}
