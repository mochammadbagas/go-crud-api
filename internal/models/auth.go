package models

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
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
	v := utils.ValidationErrors{}

	i.Username = strings.TrimSpace(i.Username)
	if i.Username == "" {
		v.Add("username", "username is required")
	} else if !usernameRegex.MatchString(i.Username) {
		v.Add("username", "username must be 3-30 characters long and can contain alphanumeric, underscore (_), or hyphen (-)")
	}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		v.Add("email", "email is required")
	} else if _, err := mail.ParseAddress(i.Email); err != nil {
		v.Add("email", "invalid email format")
	}

	if i.Password == "" {
		v.Add("password", "password is required")
	} else if len(i.Password) < 8 {
		v.Add("password", "password must be at least 8 characters long")
	}

	i.FullName = strings.TrimSpace(i.FullName)

	if v.HasErrors() {
		return v
	}
	return nil
}

// LoginInput is the payload sent during user/admin login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the login payload.
func (i *LoginInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		v.Add("email", "email is required")
	}
	if strings.TrimSpace(i.Password) == "" {
		v.Add("password", "password is required")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// RefreshTokenInput represents the request payload for refreshing tokens (when sent via JSON).
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate validates the refresh token input.
func (i *RefreshTokenInput) Validate() error {
	v := utils.ValidationErrors{}

	i.RefreshToken = strings.TrimSpace(i.RefreshToken)
	if i.RefreshToken == "" {
		v.Add("refresh_token", "refresh_token is required")
	}

	if v.HasErrors() {
		return v
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
	v := utils.ValidationErrors{}

	if strings.TrimSpace(i.OldPassword) == "" {
		v.Add("old_password", "old password is required")
	}
	if len(i.NewPassword) < 8 {
		v.Add("new_password", "new password must be at least 8 characters long")
	}
	if i.OldPassword != "" && i.OldPassword == i.NewPassword {
		v.Add("new_password", "new password cannot be the same as old password")
	}

	if v.HasErrors() {
		return v
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
