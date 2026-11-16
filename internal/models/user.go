package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Allowed user roles
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

// Allowed user account statuses
const (
	StatusActive              = "active"
	StatusInactive            = "inactive"
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
	Roles           []string   `json:"roles,omitempty"`
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
	Roles           []string   `json:"roles,omitempty"`
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
	v := utils.ValidationErrors{}

	i.FullName = strings.TrimSpace(i.FullName)
	if i.FullName == "" {
		v.Add("full_name", utils.ErrCodeRequired, "full name is required")
	} else if len(i.FullName) < 2 {
		v.Add("full_name", utils.ErrCodeTooShort, "full name must be at least 2 characters long")
	} else if len(i.FullName) > 100 {
		v.Add("full_name", utils.ErrCodeTooLong, "full name must not exceed 100 characters")
	}

	if i.Bio != nil {
		bio := strings.TrimSpace(*i.Bio)
		if len(bio) > 500 {
			v.Add("bio", utils.ErrCodeTooLong, "bio must not exceed 500 characters")
		}
		i.Bio = &bio
	}
	if i.PhoneNumber != nil {
		phone := strings.TrimSpace(*i.PhoneNumber)
		if len(phone) > 20 {
			v.Add("phone_number", utils.ErrCodeTooLong, "phone number must not exceed 20 characters")
		}
		i.PhoneNumber = &phone
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdateUsernameInput represents the payload for changing username.
type UpdateUsernameInput struct {
	Username string `json:"username"`
}

// Validate validates the new username.
func (i *UpdateUsernameInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Username = strings.TrimSpace(i.Username)
	if i.Username == "" {
		v.Add("username", utils.ErrCodeRequired, "username is required")
	} else if !utils.IsValidUsername(i.Username) {
		v.Add("username", utils.ErrCodeInvalidFormat, "username must be 3-30 characters long and can contain alphanumeric, underscore (_), or hyphen (-)")
	} else if utils.IsReservedUsername(i.Username) {
		v.Add("username", utils.ErrCodeReserved, "this username is reserved and cannot be used")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}
