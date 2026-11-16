package models

import (
	"errors"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

func TestRegisterInputValidation(t *testing.T) {
	// All fields empty
	input := RegisterInput{}
	err := input.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var valErr utils.ValidationErrors
	if !errors.As(err, &valErr) {
		t.Fatalf("expected error to be utils.ValidationErrors, got %T", err)
	}

	// Should contain username, email, password, full_name errors
	if f, ok := valErr["username"]; !ok || f.Code != utils.ErrCodeRequired {
		t.Errorf("expected username required error, got %+v", f)
	}
	if f, ok := valErr["email"]; !ok || f.Code != utils.ErrCodeRequired {
		t.Errorf("expected email required error, got %+v", f)
	}
	if f, ok := valErr["password"]; !ok || f.Code != utils.ErrCodeRequired {
		t.Errorf("expected password required error, got %+v", f)
	}
	if f, ok := valErr["full_name"]; !ok || f.Code != utils.ErrCodeRequired {
		t.Errorf("expected full_name required error, got %+v", f)
	}

	// Email without TLD should fail
	invalidEmail := RegisterInput{
		Username: "user123",
		Email:    "user@domain",
		Password: "password123",
		FullName: "User Name",
	}
	if err := invalidEmail.Validate(); err == nil {
		t.Fatal("expected error for email without TLD, got nil")
	} else if errors.As(err, &valErr) && valErr["email"].Code != utils.ErrCodeInvalidEmail {
		t.Errorf("expected INVALID_EMAIL code, got %s", valErr["email"].Code)
	}

	// Reserved username should fail
	reservedUser := RegisterInput{
		Username: "admin",
		Email:    "valid@example.com",
		Password: "password123",
		FullName: "Admin Person",
	}
	if err := reservedUser.Validate(); err == nil {
		t.Fatal("expected error for reserved username, got nil")
	} else if errors.As(err, &valErr) && valErr["username"].Code != utils.ErrCodeReserved {
		t.Errorf("expected RESERVED code, got %s", valErr["username"].Code)
	}

	// Password exceeding 128 characters should fail
	longPassword := RegisterInput{
		Username: "valid_user",
		Email:    "valid@example.com",
		Password: string(make([]byte, 129)),
		FullName: "Valid User",
	}
	if err := longPassword.Validate(); err == nil {
		t.Fatal("expected error for password exceeding 128 chars, got nil")
	} else if errors.As(err, &valErr) && valErr["password"].Code != utils.ErrCodeTooLong {
		t.Errorf("expected TOO_LONG code, got %s", valErr["password"].Code)
	}

	// Valid input
	validInput := RegisterInput{
		Username: "valid_user123",
		Email:    "test@example.com",
		Password: "strongpassword123",
		FullName: "Valid User",
	}
	if err := validInput.Validate(); err != nil {
		t.Fatalf("expected valid input to pass, got: %v", err)
	}
}

func TestLoginInputValidation(t *testing.T) {
	input := LoginInput{}
	err := input.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var valErr utils.ValidationErrors
	if !errors.As(err, &valErr) {
		t.Fatalf("expected error to be utils.ValidationErrors, got %T", err)
	}

	if _, ok := valErr["email"]; !ok {
		t.Error("expected email error")
	}
	if _, ok := valErr["password"]; !ok {
		t.Error("expected password error")
	}
}

func TestChangePasswordInputValidation(t *testing.T) {
	// Empty old password and short new password
	input := ChangePasswordInput{
		OldPassword: "",
		NewPassword: "short",
	}
	err := input.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var valErr utils.ValidationErrors
	if !errors.As(err, &valErr) {
		t.Fatalf("expected error to be utils.ValidationErrors, got %T", err)
	}

	if _, ok := valErr["old_password"]; !ok {
		t.Error("expected old_password error")
	}
	if _, ok := valErr["new_password"]; !ok {
		t.Error("expected new_password error")
	}

	// Same password
	sameInput := ChangePasswordInput{
		OldPassword: "oldpassword123",
		NewPassword: "oldpassword123",
	}
	err = sameInput.Validate()
	if err == nil {
		t.Fatal("expected error for identical password")
	}
	if !errors.As(err, &valErr) || valErr["new_password"].Code != utils.ErrCodeInvalidValue {
		t.Errorf("expected new_password error code %s for identical passwords, got %v", utils.ErrCodeInvalidValue, valErr)
	}
}

func TestCreatePostInputValidation(t *testing.T) {
	input := CreatePostInput{}
	err := input.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var valErr utils.ValidationErrors
	if !errors.As(err, &valErr) {
		t.Fatalf("expected error to be utils.ValidationErrors, got %T", err)
	}

	if _, ok := valErr["type"]; !ok {
		t.Error("expected type error")
	}
	if _, ok := valErr["title"]; !ok {
		t.Error("expected title error")
	}
	if _, ok := valErr["content"]; !ok {
		t.Error("expected content error")
	}

	// Valid input
	valid := CreatePostInput{
		Type:    TypeBlog,
		Title:   "Valid Title",
		Content: "Valid content text here",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid post input to pass, got %v", err)
	}
	if valid.Status != StatusDraft {
		t.Errorf("expected default status to be draft, got %s", valid.Status)
	}
}

func TestCreateRoleInputValidation(t *testing.T) {
	input := CreateRoleInput{}
	err := input.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var valErr utils.ValidationErrors
	if !errors.As(err, &valErr) {
		t.Fatalf("expected error to be utils.ValidationErrors, got %T", err)
	}

	if _, ok := valErr["name"]; !ok {
		t.Error("expected name error")
	}
	if _, ok := valErr["display_name"]; !ok {
		t.Error("expected display_name error")
	}
}

func TestCategoryInputValidation(t *testing.T) {
	empty := CategoryInput{}
	if err := empty.Validate(); err == nil {
		t.Fatal("expected error for empty CategoryInput")
	}

	invalidSlug := CategoryInput{
		Name: "Tech Category",
		Slug: "Invalid Slug With Spaces",
	}
	if err := invalidSlug.Validate(); err == nil {
		t.Fatal("expected error for invalid slug")
	}

	valid := CategoryInput{
		Name: "Tech Category",
		Slug: "tech-category",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid CategoryInput to pass, got %v", err)
	}
}

func TestUpdateProfileInputValidation(t *testing.T) {
	empty := UpdateProfileInput{}
	if err := empty.Validate(); err == nil {
		t.Fatal("expected error for empty full_name")
	}

	valid := UpdateProfileInput{
		FullName: "Mochammad Bagas",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid UpdateProfileInput to pass, got %v", err)
	}
}
