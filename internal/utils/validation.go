package utils

import (
	"fmt"
	"sort"
	"strings"
)

// ValidationErrors stores field-specific validation error messages.
type ValidationErrors map[string]string

// Add sets a validation error message for a given field if not already set.
func (v ValidationErrors) Add(field, message string) {
	if _, exists := v[field]; !exists {
		v[field] = message
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
		sb.WriteString(fmt.Sprintf("%s: %s", k, v[k]))
	}
	return sb.String()
}
