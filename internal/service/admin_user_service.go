package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

var (
	ErrCannotDeleteLastSuperAdmin     = errors.New("cannot delete the last active Super Admin")
	ErrCannotDeactivateLastSuperAdmin = errors.New("cannot deactivate the last active Super Admin")
	ErrCannotDemoteLastSuperAdmin     = errors.New("cannot revoke Super Admin role from the last active Super Admin")
)

type AdminUserServiceInterface interface {
	ListUsers(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error)
	GetUserDetail(ctx context.Context, id string) (*models.UserWithProfile, error)
	UpdateUser(ctx context.Context, id string, input models.AdminUpdateUserInput) (*models.UserWithProfile, error)
	UpdateUserStatus(ctx context.Context, id string, input models.UpdateUserStatusInput) (*models.UserWithProfile, error)
	AssignUserRoles(ctx context.Context, id string, input models.AssignRolesInput) (*models.UserWithProfile, error)
	DeleteUser(ctx context.Context, id string) error
}

type AdminUserService struct {
	userRepo    repository.UserRepository
	profileRepo repository.UserProfileRepository
	roleRepo    repository.RoleRepository
}

func NewAdminUserService(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	roleRepo repository.RoleRepository,
) *AdminUserService {
	return &AdminUserService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		roleRepo:    roleRepo,
	}
}

func (s *AdminUserService) ListUsers(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error) {
	return s.userRepo.ListAdmin(ctx, filter)
}

func (s *AdminUserService) GetUserDetail(ctx context.Context, id string) (*models.UserWithProfile, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	result := &models.UserWithProfile{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		Role:            user.Role,
		Roles:           user.Roles,
		Status:          user.Status,
		EmailVerifiedAt: user.EmailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	profile, err := s.profileRepo.GetByUserID(ctx, id)
	if err == nil && profile != nil {
		result.FullName = profile.FullName
		result.Bio = profile.Bio
		result.PhoneNumber = profile.PhoneNumber
		result.AvatarURL = profile.AvatarURL
	}

	return result, nil
}

func (s *AdminUserService) UpdateUser(ctx context.Context, id string, input models.AdminUpdateUserInput) (*models.UserWithProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Check email duplication if changed
	if !strings.EqualFold(user.Email, input.Email) {
		existingEmail, err := s.userRepo.GetByEmail(ctx, input.Email)
		if err == nil && existingEmail != nil && existingEmail.ID != id {
			return nil, ErrEmailExists
		}
	}

	// Check username duplication if changed
	if !strings.EqualFold(user.Username, input.Username) {
		existingUsername, err := s.userRepo.GetByUsername(ctx, input.Username)
		if err == nil && existingUsername != nil && existingUsername.ID != id {
			return nil, ErrUsernameExists
		}
	}

	if err := s.userRepo.UpdateByAdmin(ctx, id, input.Username, input.Email); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	if input.FullName != "" || input.Bio != nil || input.PhoneNumber != nil {
		profile, err := s.profileRepo.GetByUserID(ctx, id)
		if err == nil && profile != nil {
			if input.FullName != "" {
				profile.FullName = input.FullName
			}
			if input.Bio != nil {
				profile.Bio = input.Bio
			}
			if input.PhoneNumber != nil {
				profile.PhoneNumber = input.PhoneNumber
			}
			_ = s.profileRepo.Update(ctx, profile)
		} else {
			fullName := input.FullName
			if fullName == "" {
				fullName = input.Username
			}
			_ = s.profileRepo.Create(ctx, &models.UserProfile{
				UserID:      id,
				FullName:    fullName,
				Bio:         input.Bio,
				PhoneNumber: input.PhoneNumber,
			})
		}
	}

	return s.GetUserDetail(ctx, id)
}

func (s *AdminUserService) UpdateUserStatus(ctx context.Context, id string, input models.UpdateUserStatusInput) (*models.UserWithProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Safeguard: Do not deactivate or suspend the last active Super Admin
	if input.Status == models.StatusInactive || input.Status == models.StatusSuspended {
		isSuperAdmin, _ := s.userRepo.CheckUserHasRole(ctx, id, models.RoleSuperAdmin)
		if !isSuperAdmin {
			isSuperAdmin = (user.Role == models.RoleSuperAdmin)
		}

		if isSuperAdmin && user.Status == models.StatusActive {
			remaining, err := s.userRepo.CountActiveSuperAdminsExcluding(ctx, id)
			if err != nil {
				return nil, err
			}
			if remaining == 0 {
				return nil, ErrCannotDeactivateLastSuperAdmin
			}
		}
	}

	if err := s.userRepo.UpdateStatus(ctx, id, input.Status); err != nil {
		return nil, err
	}

	return s.GetUserDetail(ctx, id)
}

func (s *AdminUserService) AssignUserRoles(ctx context.Context, id string, input models.AssignRolesInput) (*models.UserWithProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Validate role IDs and check if super_admin is being assigned
	willHaveSuperAdmin := false
	for _, roleID := range input.RoleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return nil, ErrRoleNotFound
		}
		if role.Name == models.RoleSuperAdmin {
			willHaveSuperAdmin = true
		}
	}

	// Safeguard: Check if user currently has super_admin role and is being stripped of it
	isCurrentlySuperAdmin, _ := s.userRepo.CheckUserHasRole(ctx, id, models.RoleSuperAdmin)
	if !isCurrentlySuperAdmin {
		isCurrentlySuperAdmin = (user.Role == models.RoleSuperAdmin)
	}

	if isCurrentlySuperAdmin && !willHaveSuperAdmin && user.Status == models.StatusActive {
		remaining, err := s.userRepo.CountActiveSuperAdminsExcluding(ctx, id)
		if err != nil {
			return nil, err
		}
		if remaining == 0 {
			return nil, ErrCannotDemoteLastSuperAdmin
		}
	}

	if err := s.userRepo.SetRoles(ctx, id, input.RoleIDs); err != nil {
		return nil, err
	}
	middleware.ClearUserPermissionCache(id)

	return s.GetUserDetail(ctx, id)
}

func (s *AdminUserService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// Safeguard: Do not delete the last active Super Admin
	isSuperAdmin, _ := s.userRepo.CheckUserHasRole(ctx, id, models.RoleSuperAdmin)
	if !isSuperAdmin {
		isSuperAdmin = (user.Role == models.RoleSuperAdmin)
	}

	if isSuperAdmin && user.Status == models.StatusActive {
		remaining, err := s.userRepo.CountActiveSuperAdminsExcluding(ctx, id)
		if err != nil {
			return err
		}
		if remaining == 0 {
			return ErrCannotDeleteLastSuperAdmin
		}
	}

	return s.userRepo.Delete(ctx, id)
}
