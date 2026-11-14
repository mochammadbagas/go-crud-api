package models

import (
	"errors"
	"strings"
	"time"
)

// Allowed user roles
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Allowed user account statuses
const (
	StatusActive              = "active"
	StatusPendingVerification = "pending_verification"
	StatusSuspended           = "suspended"
)

// User represents the core user account entity.
type User struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserProfile represents personal profile information.
type UserProfile struct {
	UserID      string    `json:"user_id"`
	FullName    string    `json:"full_name"`
	Bio         *string   `json:"bio,omitempty"`
	PhoneNumber *string   `json:"phone_number,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserWithProfile represents the aggregate response of a user and their profile.
type UserWithProfile struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	FullName        string     `json:"full_name"`
	Bio             *string    `json:"bio,omitempty"`
	PhoneNumber     *string    `json:"phone_number,omitempty"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
}

// UpdateProfileInput represents the payload for updating personal profile details.
type UpdateProfileInput struct {
	FullName    string  `json:"full_name"`
	Bio         *string `json:"bio"`
	PhoneNumber *string `json:"phone_number"`
}

// Validate validates the profile update payload.
func (i *UpdateProfileInput) Validate() error {
	i.FullName = strings.TrimSpace(i.FullName)
	if i.FullName == "" {
		return errors.New("nama lengkap tidak boleh kosong")
	}
	if len(i.FullName) > 100 {
		return errors.New("nama lengkap maksimal 100 karakter")
	}
	if i.Bio != nil {
		bio := strings.TrimSpace(*i.Bio)
		i.Bio = &bio
	}
	if i.PhoneNumber != nil {
		phone := strings.TrimSpace(*i.PhoneNumber)
		i.PhoneNumber = &phone
	}
	return nil
}

// UpdateUsernameInput represents the payload for changing username.
type UpdateUsernameInput struct {
	Username string `json:"username"`
}

// Validate validates the new username.
func (i *UpdateUsernameInput) Validate() error {
	i.Username = strings.TrimSpace(i.Username)
	if !usernameRegex.MatchString(i.Username) {
		return errors.New("username harus 3-30 karakter alfanumerik, garis bawah (_), atau strip (-)")
	}
	return nil
}
