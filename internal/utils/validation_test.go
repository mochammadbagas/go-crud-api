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

	v.Add("email", ErrCodeInvalidEmail, "invalid email format")
	v.Add("password", ErrCodeTooShort, "password must be at least 8 characters long")

	if !v.HasErrors() {
		t.Fatal("expected ValidationErrors to have errors")
	}

	if len(v) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(v))
	}

	// Duplicate add should preserve first error
	v.Add("email", ErrCodeRequired, "duplicate email error ignored")
	if v["email"].Code != ErrCodeInvalidEmail || v["email"].Message != "invalid email format" {
		t.Fatalf("expected preserved first error, got %+v", v["email"])
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
	if target["email"].Code != ErrCodeInvalidEmail || target["email"].Message != "invalid email format" {
		t.Fatalf("expected target['email'] to match, got %+v", target["email"])
	}
}

func TestIsValidEmail(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"user.name+tag@sub.domain.co.id",
		"developer@domain.io",
		"alice123@company.org",
	}
	for _, e := range validEmails {
		if !IsValidEmail(e) {
			t.Errorf("expected valid email %q to pass", e)
		}
	}

	invalidEmails := []string{
		"",
		"   ",
		"invalid",
		"@domain.com",
		"user@",
		"user@domain",          // No TLD!
		"user@localhost",       // Localhost without TLD
		"user@.com",
		"user..name@domain.com", // Consecutive dots
		"user@domain..com",     // Consecutive dots in domain
		"user name@domain.com", // Spaces
		"user@domain.c",        // Single-character TLD
	}
	for _, e := range invalidEmails {
		if IsValidEmail(e) {
			t.Errorf("expected invalid email %q to fail", e)
		}
	}
}

func TestIsReservedUsername(t *testing.T) {
	reserved := []string{"admin", "ADMIN", " Administrator ", "root", "system", "api", "guest", "test"}
	for _, u := range reserved {
		if !IsReservedUsername(u) {
			t.Errorf("expected %q to be reserved", u)
		}
	}

	allowed := []string{"johndoe", "alice_wonder", "alex-smith", "developer99"}
	for _, u := range allowed {
		if IsReservedUsername(u) {
			t.Errorf("expected %q to be allowed", u)
		}
	}
}

func TestIsValidSlug(t *testing.T) {
	valid := []string{"hello-world", "go-starter-kit-2026", "news"}
	for _, s := range valid {
		if !IsValidSlug(s) {
			t.Errorf("expected slug %q to be valid", s)
		}
	}

	invalid := []string{"", "Hello World", "hello_world", "slug--slug", "-slug", "slug-"}
	for _, s := range invalid {
		if IsValidSlug(s) {
			t.Errorf("expected slug %q to be invalid", s)
		}
	}
}
