package models

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	roleNameRegex       = regexp.MustCompile(`^[a-z0-9_-]{2,50}$`)
	permissionNameRegex = regexp.MustCompile(`^[a-z0-9_]{2,40}\.[a-z0-9_]{2,40}$`)
)

// Role represents a user role in the RBAC system.
type Role struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description *string    `json:"description,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Permission represents a granular permission in the RBAC system.
type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateRoleInput represents the payload for creating a new role.
type CreateRoleInput struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

// Validate validates CreateRoleInput.
func (i *CreateRoleInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Name = strings.TrimSpace(strings.ToLower(i.Name))
	if i.Name == "" {
		v.Add("name", "role name is required")
	} else if !roleNameRegex.MatchString(i.Name) {
		v.Add("name", "role name must be 2-50 lowercase alphanumeric characters, underscores (_), or hyphens (-)")
	}

	i.DisplayName = strings.TrimSpace(i.DisplayName)
	if i.DisplayName == "" {
		v.Add("display_name", "display name cannot be empty")
	} else if len(i.DisplayName) > 100 {
		v.Add("display_name", "display name must not exceed 100 characters")
	}

	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdateRoleInput represents the payload for updating an existing role.
type UpdateRoleInput struct {
	DisplayName string  `json:"display_name"`
	Description *string `json:"description"`
}

// Validate validates UpdateRoleInput.
func (i *UpdateRoleInput) Validate() error {
	v := utils.ValidationErrors{}

	i.DisplayName = strings.TrimSpace(i.DisplayName)
	if i.DisplayName == "" {
		v.Add("display_name", "display name cannot be empty")
	} else if len(i.DisplayName) > 100 {
		v.Add("display_name", "display name must not exceed 100 characters")
	}

	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// AssignPermissionsInput represents the payload for synchronizing role permissions.
type AssignPermissionsInput struct {
	PermissionIDs []string `json:"permission_ids"`
}

// CreatePermissionInput represents the payload for creating a master permission.
type CreatePermissionInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// Validate validates CreatePermissionInput.
func (i *CreatePermissionInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Name = strings.TrimSpace(strings.ToLower(i.Name))
	if i.Name == "" {
		v.Add("name", "permission name is required")
	} else if !permissionNameRegex.MatchString(i.Name) {
		v.Add("name", "permission name must follow 'resource.action' format (e.g. 'users.read', 'roles.create')")
	}

	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdatePermissionInput represents the payload for updating a master permission.
type UpdatePermissionInput struct {
	Description *string `json:"description"`
}

// Validate validates UpdatePermissionInput.
func (i *UpdatePermissionInput) Validate() error {
	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
	}
	return nil
}

// AssignRolesInput represents the payload for assigning roles to a user.
type AssignRolesInput struct {
	RoleIDs []string `json:"role_ids"`
}

// Validate validates AssignRolesInput.
func (i *AssignRolesInput) Validate() error {
	v := utils.ValidationErrors{}

	if len(i.RoleIDs) == 0 {
		v.Add("role_ids", "user must have at least one role")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdateUserStatusInput represents the payload for changing a user's account status.
type UpdateUserStatusInput struct {
	Status string `json:"status"`
}

// Validate validates UpdateUserStatusInput.
func (i *UpdateUserStatusInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Status = strings.TrimSpace(strings.ToLower(i.Status))
	switch i.Status {
	case StatusActive, StatusInactive, StatusPendingVerification, StatusSuspended:
		return nil
	default:
		v.Add("status", "invalid status, choose between: active, inactive, pending_verification, suspended")
		return v
	}
}

// AdminUpdateUserInput represents the payload for administrative update of a user.
type AdminUpdateUserInput struct {
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	Bio         *string `json:"bio"`
	PhoneNumber *string `json:"phone_number"`
}

// Validate validates AdminUpdateUserInput.
func (i *AdminUpdateUserInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Username = strings.TrimSpace(i.Username)
	if i.Username == "" {
		v.Add("username", "username is required")
	} else if !usernameRegex.MatchString(i.Username) {
		v.Add("username", "username must be 3-30 characters long and can contain alphanumeric, underscore (_), or hyphen (-)")
	}

	i.Email = strings.TrimSpace(strings.ToLower(i.Email))
	if i.Email == "" {
		v.Add("email", "email is required")
	} else if _, err := mail.ParseAddress(i.Email); err != nil {
		v.Add("email", "invalid email format")
	}

	i.FullName = strings.TrimSpace(i.FullName)
	if i.FullName != "" && len(i.FullName) > 100 {
		v.Add("full_name", "full name must not exceed 100 characters")
	}

	if i.Bio != nil {
		bio := strings.TrimSpace(*i.Bio)
		i.Bio = &bio
	}
	if i.PhoneNumber != nil {
		phone := strings.TrimSpace(*i.PhoneNumber)
		i.PhoneNumber = &phone
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// AdminUserFilter represents query params for listing users.
type AdminUserFilter struct {
	Search string
	Status string
	Role   string
	Page   int
	Limit  int
}
