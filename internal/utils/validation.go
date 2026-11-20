package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// emailRegex enforces standard email format: local-part@domain.tld with at least 2 chars TLD
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

	// slugRegex enforces URL-safe lowercase alphanumeric and hyphens
	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	// usernameRegex enforces 3-30 characters alphanumeric, underscore, or hyphen
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

	// reservedUsernames protects critical system identifiers and core route paths from collision
	reservedUsernames = map[string]struct{}{
		"admin": {}, "administrator": {}, "root": {}, "system": {},
		"superuser": {}, "support": {}, "security": {}, "api": {},
		"null": {}, "undefined": {}, "guest": {}, "test": {},
		"me": {}, "auth": {}, "users": {}, "posts": {},
		"articles": {}, "categories": {}, "tags": {}, "media": {},
		"dashboard": {}, "settings": {}, "login": {}, "register": {},
		"logout": {}, "analytics": {}, "roles": {}, "permissions": {},
	}
)

// IsValidEmail verifies an email string against production specifications (RFC 5321 / RFC 5322 with TLD).
func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) < 6 || len(email) > 254 {
		return false
	}
	if strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	if strings.Contains(email, "..") {
		return false
	}
	atIdx := strings.LastIndex(email, "@")
	if atIdx <= 0 || atIdx >= len(email)-3 {
		return false
	}
	// Local-part cannot exceed 64 octets (RFC 5321)
	if len(email[:atIdx]) > 64 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsReservedUsername checks whether the given username is reserved by the system.
func IsReservedUsername(username string) bool {
	clean := strings.ToLower(strings.TrimSpace(username))
	_, exists := reservedUsernames[clean]
	return exists
}

// IsValidUsername checks if a username satisfies alphanumeric, length, and format rules.
func IsValidUsername(username string) bool {
	clean := strings.TrimSpace(username)
	return usernameRegex.MatchString(clean)
}

// IsValidSlug checks if a slug is a valid URL-safe string.
func IsValidSlug(slug string) bool {
	clean := strings.TrimSpace(slug)
	if len(clean) == 0 || len(clean) > 255 {
		return false
	}
	return slugRegex.MatchString(clean)
}

// Standard machine-readable validation error codes.
const (
	ErrCodeRequired      = "REQUIRED"
	ErrCodeTooShort      = "TOO_SHORT"
	ErrCodeTooLong       = "TOO_LONG"
	ErrCodeInvalidFormat = "INVALID_FORMAT"
	ErrCodeInvalidEmail  = "INVALID_EMAIL"
	ErrCodeReserved      = "RESERVED"
	ErrCodeInvalidID     = "INVALID_ID"
	ErrCodeInvalidStatus = "INVALID_STATUS"
	ErrCodeInvalidValue  = "INVALID_VALUE"
)

// FieldError represents a structured validation error with a machine-readable code
// and a human-readable message for client rendering and internationalization (i18n).
type FieldError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (f FieldError) String() string {
	return f.Message
}

// ValidationErrors stores field-specific validation error details.
type ValidationErrors map[string]FieldError

// Add sets a validation error code and message for a given field if not already set.
func (v ValidationErrors) Add(field, code, message string) {
	if _, exists := v[field]; !exists {
		v[field] = FieldError{
			Code:    code,
			Message: message,
		}
	}
}

// HasErrors returns true if there is at least one validation error.
func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

// Error implements the standard error interface.
// It joins all errors in alphabetical order of field names for deterministic strings.
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", k, v[k].Message))
	}
	return sb.String()
}
