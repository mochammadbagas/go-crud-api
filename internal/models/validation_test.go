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

	// Should contain username, email, password errors
	if _, ok := valErr["username"]; !ok {
		t.Error("expected username error")
	}
	if _, ok := valErr["email"]; !ok {
		t.Error("expected email error")
	}
	if _, ok := valErr["password"]; !ok {
		t.Error("expected password error")
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
	if !errors.As(err, &valErr) || valErr["new_password"] == "" {
		t.Errorf("expected new_password error for identical passwords, got %v", valErr)
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
