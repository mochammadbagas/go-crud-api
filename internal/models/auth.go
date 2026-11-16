package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
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
		v.Add("username", utils.ErrCodeRequired, "username is required")
	} else if !utils.IsValidUsername(i.Username) {
		v.Add("username", utils.ErrCodeInvalidFormat, "username must be 3-30 characters long and can contain alphanumeric, underscore (_), or hyphen (-)")
	} else if utils.IsReservedUsername(i.Username) {
		v.Add("username", utils.ErrCodeReserved, "this username is reserved and cannot be registered")
	}

	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	if i.Email == "" {
		v.Add("email", utils.ErrCodeRequired, "email is required")
	} else if !utils.IsValidEmail(i.Email) {
		v.Add("email", utils.ErrCodeInvalidEmail, "invalid email format")
	}

	if i.Password == "" {
		v.Add("password", utils.ErrCodeRequired, "password is required")
	} else if len(i.Password) < 8 {
		v.Add("password", utils.ErrCodeTooShort, "password must be at least 8 characters long")
	} else if len(i.Password) > 128 {
		v.Add("password", utils.ErrCodeTooLong, "password must not exceed 128 characters")
	}

	i.FullName = strings.TrimSpace(i.FullName)
	if i.FullName == "" {
		v.Add("full_name", utils.ErrCodeRequired, "full name is required")
	} else if len(i.FullName) < 2 {
		v.Add("full_name", utils.ErrCodeTooShort, "full name must be at least 2 characters long")
	} else if len(i.FullName) > 100 {
		v.Add("full_name", utils.ErrCodeTooLong, "full name must not exceed 100 characters")
	}

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
		v.Add("email", utils.ErrCodeRequired, "email is required")
	} else if !utils.IsValidEmail(i.Email) {
		v.Add("email", utils.ErrCodeInvalidEmail, "invalid email format")
	}

	if strings.TrimSpace(i.Password) == "" {
		v.Add("password", utils.ErrCodeRequired, "password is required")
	} else if len(i.Password) > 128 {
		v.Add("password", utils.ErrCodeTooLong, "password must not exceed 128 characters")
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
		v.Add("refresh_token", utils.ErrCodeRequired, "refresh_token is required")
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
		v.Add("old_password", utils.ErrCodeRequired, "old password is required")
	}
	if len(i.NewPassword) < 8 {
		v.Add("new_password", utils.ErrCodeTooShort, "new password must be at least 8 characters long")
	} else if len(i.NewPassword) > 128 {
		v.Add("new_password", utils.ErrCodeTooLong, "new password must not exceed 128 characters")
	}
	if i.OldPassword != "" && i.OldPassword == i.NewPassword {
		v.Add("new_password", utils.ErrCodeInvalidValue, "new password cannot be the same as old password")
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
