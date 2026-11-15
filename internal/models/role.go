package models

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
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
	i.Name = strings.TrimSpace(strings.ToLower(i.Name))
	if !roleNameRegex.MatchString(i.Name) {
		return errors.New("nama role harus 2-50 karakter huruf kecil, angka, garis bawah (_), atau strip (-)")
	}
	i.DisplayName = strings.TrimSpace(i.DisplayName)
	if i.DisplayName == "" {
		return errors.New("nama tampilan (display_name) tidak boleh kosong")
	}
	if len(i.DisplayName) > 100 {
		return errors.New("nama tampilan maksimal 100 karakter")
	}
	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
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
	i.DisplayName = strings.TrimSpace(i.DisplayName)
	if i.DisplayName == "" {
		return errors.New("nama tampilan (display_name) tidak boleh kosong")
	}
	if len(i.DisplayName) > 100 {
		return errors.New("nama tampilan maksimal 100 karakter")
	}
	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
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
	i.Name = strings.TrimSpace(strings.ToLower(i.Name))
	if !permissionNameRegex.MatchString(i.Name) {
		return errors.New("format nama permission harus 'resource.action' (contoh: 'users.read', 'roles.create')")
	}
	if i.Description != nil {
		desc := strings.TrimSpace(*i.Description)
		i.Description = &desc
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
	if len(i.RoleIDs) == 0 {
		return errors.New("pengguna minimal harus memiliki satu role")
	}
	return nil
}

// UpdateUserStatusInput represents the payload for changing a user's account status.
type UpdateUserStatusInput struct {
	Status string `json:"status"`
}

// Validate validates UpdateUserStatusInput.
func (i *UpdateUserStatusInput) Validate() error {
	i.Status = strings.TrimSpace(strings.ToLower(i.Status))
	switch i.Status {
	case StatusActive, StatusInactive, StatusPendingVerification, StatusSuspended:
		return nil
	default:
		return errors.New("status tidak valid, pilih antara: active, inactive, pending_verification, suspended")
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
	i.Username = strings.TrimSpace(i.Username)
	if !usernameRegex.MatchString(i.Username) {
		return errors.New("username harus 3-30 karakter alfanumerik, garis bawah (_), atau strip (-)")
	}
	i.Email = strings.TrimSpace(strings.ToLower(i.Email))
	if _, err := mail.ParseAddress(i.Email); err != nil {
		return errors.New("format email tidak valid")
	}
	i.FullName = strings.TrimSpace(i.FullName)
	if i.FullName != "" && len(i.FullName) > 100 {
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

// AdminUserFilter represents query params for listing users.
type AdminUserFilter struct {
	Search string
	Status string
	Role   string
	Page   int
	Limit  int
}
