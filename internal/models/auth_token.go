package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Allowed auth token types
const (
	TokenTypePasswordReset     = "password_reset"
	TokenTypeEmailVerification = "email_verification"
	TokenTypeEmailChange       = "email_change"
)

// AuthToken represents a single-use token record stored in the database.
type AuthToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	TokenType string    `json:"token_type"`
	Payload   *string   `json:"payload,omitempty"`
	IsUsed    bool      `json:"is_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ForgotPasswordInput represents request payload for requesting password reset.
type ForgotPasswordInput struct {
	Email string `json:"email"`
}

func (i *ForgotPasswordInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		v.Add("email", utils.ErrCodeRequired, "email is required")
	} else if !utils.IsValidEmail(i.Email) {
		v.Add("email", utils.ErrCodeInvalidEmail, "invalid email format")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// ResetPasswordInput represents request payload for resetting password.
type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (i *ResetPasswordInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		v.Add("token", utils.ErrCodeRequired, "password reset token is required")
	}
	if i.NewPassword == "" {
		v.Add("new_password", utils.ErrCodeRequired, "new password is required")
	} else if len(i.NewPassword) < 8 {
		v.Add("new_password", utils.ErrCodeTooShort, "new password must be at least 8 characters long")
	} else if len(i.NewPassword) > 128 {
		v.Add("new_password", utils.ErrCodeTooLong, "new password must not exceed 128 characters")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// VerifyEmailInput represents request payload for email verification.
type VerifyEmailInput struct {
	Token string `json:"token"`
}

func (i *VerifyEmailInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		v.Add("token", utils.ErrCodeRequired, "verification token is required")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// ResendVerificationInput represents request payload for resending verification email.
type ResendVerificationInput struct {
	Email string `json:"email"`
}

func (i *ResendVerificationInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		v.Add("email", utils.ErrCodeRequired, "email is required")
	} else if !utils.IsValidEmail(i.Email) {
		v.Add("email", utils.ErrCodeInvalidEmail, "invalid email format")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// RequestEmailChangeInput represents request payload for changing email.
type RequestEmailChangeInput struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

func (i *RequestEmailChangeInput) Validate() error {
	v := utils.ValidationErrors{}

	i.NewEmail = strings.ToLower(strings.TrimSpace(i.NewEmail))
	if i.NewEmail == "" {
		v.Add("new_email", utils.ErrCodeRequired, "new email is required")
	} else if !utils.IsValidEmail(i.NewEmail) {
		v.Add("new_email", utils.ErrCodeInvalidEmail, "invalid new email format")
	}

	if strings.TrimSpace(i.Password) == "" {
		v.Add("password", utils.ErrCodeRequired, "current password is required for security confirmation")
	} else if len(i.Password) > 128 {
		v.Add("password", utils.ErrCodeTooLong, "password must not exceed 128 characters")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// ConfirmEmailChangeInput represents request payload for confirming email change.
type ConfirmEmailChangeInput struct {
	Token string `json:"token"`
}

func (i *ConfirmEmailChangeInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		v.Add("token", utils.ErrCodeRequired, "confirmation token is required")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}
