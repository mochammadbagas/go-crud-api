package models

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)
)

// TokenResponse represents the hybrid token response sent to clients.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RegisterInput represents the input for user registration.
type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// Validate checks registration business rules.
func (i *RegisterInput) Validate() error {
	i.Username = strings.TrimSpace(i.Username)
	if !usernameRegex.MatchString(i.Username) {
		return errors.New("username harus 3-30 karakter alfanumerik, garis bawah (_), atau strip (-)")
	}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if _, err := mail.ParseAddress(i.Email); err != nil {
		return errors.New("format email tidak valid")
	}

	if len(i.Password) < 8 {
		return errors.New("password minimal 8 karakter")
	}

	i.FullName = strings.TrimSpace(i.FullName)
	return nil
}

// LoginInput is the payload sent during user/admin login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the login payload.
func (i *LoginInput) Validate() error {
	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		return errors.New("email wajib diisi")
	}
	if strings.TrimSpace(i.Password) == "" {
		return errors.New("password wajib diisi")
	}
	return nil
}

// RefreshTokenInput represents the request payload for refreshing tokens (when sent via JSON).
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate validates the refresh token input.
func (i *RefreshTokenInput) Validate() error {
	i.RefreshToken = strings.TrimSpace(i.RefreshToken)
	if i.RefreshToken == "" {
		return errors.New("refresh_token wajib disertakan")
	}
	return nil
}

// ChangePasswordInput represents the request payload for changing password.
type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Validate validates the change password payload.
func (i *ChangePasswordInput) Validate() error {
	if strings.TrimSpace(i.OldPassword) == "" {
		return errors.New("password lama wajib diisi")
	}
	if len(i.NewPassword) < 8 {
		return errors.New("password baru minimal 8 karakter")
	}
	if i.OldPassword == i.NewPassword {
		return errors.New("password baru tidak boleh sama dengan password lama")
	}
	return nil
}

// RefreshToken represents a refresh token record stored in the database.
type RefreshToken struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TokenHash   string    `json:"-"`
	TokenFamily string    `json:"-"`
	IsRevoked   bool      `json:"is_revoked"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	UserAgent   *string   `json:"user_agent,omitempty"`
	IPAddress   *string   `json:"ip_address,omitempty"`
}
