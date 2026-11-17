package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// DefaultMaxBodyBytes specifies the maximum request body size permitted for JSON parsing (1 MB).
const DefaultMaxBodyBytes int64 = 1 << 20

// ErrPayloadTooLarge is returned when the request body exceeds the maximum permitted size.
var ErrPayloadTooLarge = errors.New("request payload too large")

// DecodeJSON decodes the request body into target with a maximum size limit.
// If maxBytes is 0 or negative, DefaultMaxBodyBytes (1 MB) is used.
func DecodeJSON(w http.ResponseWriter, r *http.Request, target interface{}, maxBytes ...int64) error {
	limit := DefaultMaxBodyBytes
	if len(maxBytes) > 0 && maxBytes[0] > 0 {
		limit = maxBytes[0]
	}

	if w != nil {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	} else {
		r.Body = io.NopCloser(io.LimitReader(r.Body, limit))
	}
	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return ErrPayloadTooLarge
		}
		if errors.Is(err, io.EOF) {
			return errors.New("request body cannot be empty")
		}
		return fmt.Errorf("invalid request body format: %w", err)
	}

	return nil
}

// HandleDecodeError standardizes HTTP responses when JSON payload parsing fails.
func HandleDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrPayloadTooLarge) {
		ErrorCode(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request payload exceeds maximum allowed size (1 MB)")
		return
	}
	ErrorCode(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", err.Error())
}
