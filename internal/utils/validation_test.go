package utils

import (
	"errors"
	"testing"
)

func TestValidationErrors(t *testing.T) {
	v := ValidationErrors{}
	if v.HasErrors() {
		t.Fatal("expected empty ValidationErrors to have no errors")
	}
	if v.Error() != "" {
		t.Fatalf("expected empty error string, got %q", v.Error())
	}

	v.Add("email", "invalid email format")
	v.Add("password", "password must be at least 8 characters long")

	if !v.HasErrors() {
		t.Fatal("expected ValidationErrors to have errors")
	}

	if len(v) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(v))
	}

	// Duplicate add should preserve first error
	v.Add("email", "duplicate email error ignored")
	if v["email"] != "invalid email format" {
		t.Fatalf("expected preserved first error, got %q", v["email"])
	}

	// Test alphabetical order in Error()
	expectedErrStr := "email: invalid email format; password: password must be at least 8 characters long"
	if v.Error() != expectedErrStr {
		t.Fatalf("expected %q, got %q", expectedErrStr, v.Error())
	}

	// Test errors.As compatibility
	var target ValidationErrors
	var asErr error = v
	if !errors.As(asErr, &target) {
		t.Fatal("expected errors.As to succeed with ValidationErrors")
	}
	if target["email"] != "invalid email format" {
		t.Fatalf("expected target['email'] to match, got %q", target["email"])
	}
}
