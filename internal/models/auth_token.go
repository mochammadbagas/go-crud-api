package models

import (
	"errors"
	"net/mail"
	"strings"
	"time"
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
	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		return errors.New("email wajib diisi")
	}
	if _, err := mail.ParseAddress(i.Email); err != nil {
		return errors.New("format email tidak valid")
	}
	return nil
}

// ResetPasswordInput represents request payload for resetting password.
type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (i *ResetPasswordInput) Validate() error {
	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		return errors.New("token reset password wajib disertakan")
	}
	if len(i.NewPassword) < 8 {
		return errors.New("password baru minimal 8 karakter")
	}
	return nil
}

// VerifyEmailInput represents request payload for email verification.
type VerifyEmailInput struct {
	Token string `json:"token"`
}

func (i *VerifyEmailInput) Validate() error {
	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		return errors.New("token verifikasi wajib disertakan")
	}
	return nil
}

// ResendVerificationInput represents request payload for resending verification email.
type ResendVerificationInput struct {
	Email string `json:"email"`
}

func (i *ResendVerificationInput) Validate() error {
	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		return errors.New("email wajib diisi")
	}
	if _, err := mail.ParseAddress(i.Email); err != nil {
		return errors.New("format email tidak valid")
	}
	return nil
}

// RequestEmailChangeInput represents request payload for changing email.
type RequestEmailChangeInput struct {
	NewEmail string `json:"new_email"`
	Password string `json:"password"`
}

func (i *RequestEmailChangeInput) Validate() error {
	i.NewEmail = strings.ToLower(strings.TrimSpace(i.NewEmail))
	if i.NewEmail == "" {
		return errors.New("email baru wajib diisi")
	}
	if _, err := mail.ParseAddress(i.NewEmail); err != nil {
		return errors.New("format email baru tidak valid")
	}
	if strings.TrimSpace(i.Password) == "" {
		return errors.New("password saat ini wajib diisi untuk konfirmasi keamanan")
	}
	return nil
}

// ConfirmEmailChangeInput represents request payload for confirming email change.
type ConfirmEmailChangeInput struct {
	Token string `json:"token"`
}

func (i *ConfirmEmailChangeInput) Validate() error {
	i.Token = strings.TrimSpace(i.Token)
	if i.Token == "" {
		return errors.New("token konfirmasi wajib disertakan")
	}
	return nil
}
