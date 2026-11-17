package utils

import (
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrInvalidULID = errors.New("invalid ULID format, must be 26 Crockford Base32 characters")
	entropyPool    = ulid.Monotonic(rand.Reader, 0)
	entropyPoolMu  sync.Mutex
)

// NewULID generates a new cryptographically secure and monotonic ULID string.
func NewULID() string {
	t := time.Now().UTC()
	entropyPoolMu.Lock()
	defer entropyPoolMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(t), entropyPool).String()
}

// IsValidULID checks if a string is a valid ULID.
func IsValidULID(id string) bool {
	if len(id) != 26 {
		return false
	}
	_, err := ulid.Parse(id)
	return err == nil
}

// ParseULID parses and validates a ULID string.
func ParseULID(id string) (ulid.ULID, error) {
	parsed, err := ulid.Parse(id)
	if err != nil {
		return ulid.ULID{}, ErrInvalidULID
	}
	return parsed, nil
}
