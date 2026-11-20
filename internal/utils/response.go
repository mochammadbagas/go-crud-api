package utils

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// ErrorResponse defines the standard error response JSON format.
type ErrorResponse struct {
	Code    string                `json:"code"`
	Error   string                `json:"error"`
	Message string                `json:"message"`
	Errors  map[string]FieldError `json:"errors,omitempty"`
}

// DefaultErrorCode maps standard HTTP status codes to machine-readable string codes.
func DefaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusGone:
		return "GONE"
	case http.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	case http.StatusTooManyRequests:
		return "RATE_LIMIT_EXCEEDED"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		if status >= 400 && status < 500 {
			return "CLIENT_ERROR"
		}
		return "SERVER_ERROR"
	}
}

// JSON writes a JSON response with status code and Content-Type header.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a JSON error response with an inferred code based on status code.
func Error(w http.ResponseWriter, status int, message string) {
	ErrorCode(w, status, DefaultErrorCode(status), message)
}

// ErrorCode writes a JSON error response with an explicit machine-readable code.
func ErrorCode(w http.ResponseWriter, status int, code, message string) {
	if code == "" {
		code = DefaultErrorCode(status)
	}
	JSON(w, status, ErrorResponse{
		Code:    code,
		Error:   message,
		Message: message,
	})
}

// ValidationError writes a JSON response for validation errors.
// If err is (or wraps) ValidationErrors, it outputs HTTP 400 with field-level details.
// If err is any other unhandled error, it masks the internal details, logs the error,
// and returns standard HTTP 500 INTERNAL_SERVER_ERROR to prevent CWE-209 information disclosure.
func ValidationError(w http.ResponseWriter, err error) {
	var valErr ValidationErrors
	if errors.As(err, &valErr) {
		JSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "VALIDATION_FAILED",
			Error:   "validation failed",
			Message: "validation failed",
			Errors:  valErr,
		})
		return
	}
	slog.Error("Unexpected internal error masked from client", slog.Any("error", err))
	ErrorCode(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an internal server error occurred")
}

// ValidationErrorContext is the context-aware variant of ValidationError that attaches request tracing attributes.
func ValidationErrorContext(w http.ResponseWriter, r *http.Request, err error) {
	var valErr ValidationErrors
	if errors.As(err, &valErr) {
		JSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "VALIDATION_FAILED",
			Error:   "validation failed",
			Message: "validation failed",
			Errors:  valErr,
		})
		return
	}
	slog.ErrorContext(r.Context(), "Unexpected internal error masked from client", slog.Any("error", err))
	ErrorCode(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an internal server error occurred")
}
