package service

import (
	"context"
	"errors"

	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

var (
	ErrRoleNotFound           = errors.New("role not found")
	ErrRoleNameExists         = errors.New("role name is already in use")
	ErrCannotDeleteSystemRole = errors.New("system roles cannot be deleted")
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrPermissionNameExists   = errors.New("permission name is already in use")
)

type RBACServiceInterface interface {
	CreateRole(ctx context.Context, input models.CreateRoleInput) (*models.Role, error)
	GetRoleByID(ctx context.Context, id string) (*models.Role, error)
	ListRoles(ctx context.Context) ([]*models.Role, error)
	UpdateRole(ctx context.Context, id string, input models.UpdateRoleInput) (*models.Role, error)
	DeleteRole(ctx context.Context, id string) error
	AssignPermissionsToRole(ctx context.Context, roleID string, input models.AssignPermissionsInput) error

	CreatePermission(ctx context.Context, input models.CreatePermissionInput) (*models.Permission, error)
	GetPermissionByID(ctx context.Context, id string) (*models.Permission, error)
	ListPermissions(ctx context.Context) ([]*models.Permission, error)
	UpdatePermission(ctx context.Context, id string, input models.UpdatePermissionInput) (*models.Permission, error)
	DeletePermission(ctx context.Context, id string) error
}

type RBACService struct {
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

func NewRBACService(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *RBACService {
	return &RBACService{
		roleRepo: roleRepo,
		permRepo: permRepo,
	}
}

func (s *RBACService) CreateRole(ctx context.Context, input models.CreateRoleInput) (*models.Role, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.roleRepo.GetByName(ctx, input.Name)
	if err == nil && existing != nil {
		return nil, ErrRoleNameExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	role := &models.Role{
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Permissions: input.Permissions,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrRoleNameExists
		}
		return nil, err
	}

	return s.roleRepo.GetByID(ctx, role.ID)
}

func (s *RBACService) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]*models.Role, error) {
	return s.roleRepo.List(ctx)
}

func (s *RBACService) UpdateRole(ctx context.Context, id string, input models.UpdateRoleInput) (*models.Role, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	_, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	if err := s.roleRepo.Update(ctx, id, input.DisplayName, input.Description); err != nil {
		return nil, err
	}

	return s.roleRepo.GetByID(ctx, id)
}

func (s *RBACService) DeleteRole(ctx context.Context, id string) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	// Protect default system roles
	if role.Name == models.RoleSuperAdmin || role.Name == models.RoleAdmin || role.Name == models.RoleUser {
		return ErrCannotDeleteSystemRole
	}

	return s.roleRepo.Delete(ctx, id)
}

func (s *RBACService) AssignPermissionsToRole(ctx context.Context, roleID string, input models.AssignPermissionsInput) error {
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRoleNotFound
		}
		return err
	}

	if err := s.roleRepo.AssignPermissions(ctx, roleID, input.PermissionIDs); err != nil {
		return err
	}
	middleware.ClearUserPermissionCache("")
	return nil
}

func (s *RBACService) CreatePermission(ctx context.Context, input models.CreatePermissionInput) (*models.Permission, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.permRepo.GetByName(ctx, input.Name)
	if err == nil && existing != nil {
		return nil, ErrPermissionNameExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	perm := &models.Permission{
		Name:        input.Name,
		Description: input.Description,
	}

	if err := s.permRepo.Create(ctx, perm); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrPermissionNameExists
		}
		return nil, err
	}

	return s.permRepo.GetByID(ctx, perm.ID)
}

func (s *RBACService) GetPermissionByID(ctx context.Context, id string) (*models.Permission, error) {
	perm, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPermissionNotFound
		}
		return nil, err
	}
	return perm, nil
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]*models.Permission, error) {
	return s.permRepo.List(ctx)
}

func (s *RBACService) UpdatePermission(ctx context.Context, id string, input models.UpdatePermissionInput) (*models.Permission, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	_, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPermissionNotFound
		}
		return nil, err
	}

	if err := s.permRepo.Update(ctx, id, input.Description); err != nil {
		return nil, err
	}

	return s.permRepo.GetByID(ctx, id)
}

func (s *RBACService) DeletePermission(ctx context.Context, id string) error {
	_, err := s.permRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrPermissionNotFound
		}
		return err
	}

	return s.permRepo.Delete(ctx, id)
}
