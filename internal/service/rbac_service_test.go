package service

import (
	"context"
	"errors"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type mockRoleRepo struct {
	roles   map[string]*models.Role
	byName  map[string]*models.Role
	permIDs map[string][]string
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:   make(map[string]*models.Role),
		byName:  make(map[string]*models.Role),
		permIDs: make(map[string][]string),
	}
}

func (r *mockRoleRepo) Create(ctx context.Context, role *models.Role) error {
	if role.ID == "" {
		role.ID = utils.NewULID()
	}
	r.roles[role.ID] = role
	r.byName[role.Name] = role
	return nil
}

func (r *mockRoleRepo) GetByID(ctx context.Context, id string) (*models.Role, error) {
	if role, ok := r.roles[id]; ok {
		return role, nil
	}
	return nil, repository.ErrNotFound
}

func (r *mockRoleRepo) GetByName(ctx context.Context, name string) (*models.Role, error) {
	if role, ok := r.byName[name]; ok {
		return role, nil
	}
	return nil, repository.ErrNotFound
}

func (r *mockRoleRepo) List(ctx context.Context) ([]*models.Role, error) {
	var list []*models.Role
	for _, role := range r.roles {
		list = append(list, role)
	}
	return list, nil
}

func (r *mockRoleRepo) Update(ctx context.Context, id string, displayName string, description *string) error {
	if role, ok := r.roles[id]; ok {
		role.DisplayName = displayName
		role.Description = description
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockRoleRepo) Delete(ctx context.Context, id string) error {
	if role, ok := r.roles[id]; ok {
		delete(r.byName, role.Name)
		delete(r.roles, id)
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockRoleRepo) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	r.permIDs[roleID] = permissionIDs
	return nil
}

func (r *mockRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*models.Permission, error) {
	return []*models.Permission{}, nil
}

func (r *mockRoleRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.roles)), nil
}

type mockPermRepo struct {
	perms  map[string]*models.Permission
	byName map[string]*models.Permission
}

func newMockPermRepo() *mockPermRepo {
	return &mockPermRepo{
		perms:  make(map[string]*models.Permission),
		byName: make(map[string]*models.Permission),
	}
}

func (r *mockPermRepo) Create(ctx context.Context, perm *models.Permission) error {
	if perm.ID == "" {
		perm.ID = utils.NewULID()
	}
	r.perms[perm.ID] = perm
	r.byName[perm.Name] = perm
	return nil
}

func (r *mockPermRepo) GetByID(ctx context.Context, id string) (*models.Permission, error) {
	if p, ok := r.perms[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *mockPermRepo) GetByName(ctx context.Context, name string) (*models.Permission, error) {
	if p, ok := r.byName[name]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *mockPermRepo) List(ctx context.Context) ([]*models.Permission, error) {
	var list []*models.Permission
	for _, p := range r.perms {
		list = append(list, p)
	}
	return list, nil
}

func (r *mockPermRepo) Update(ctx context.Context, id string, description *string) error {
	if p, ok := r.perms[id]; ok {
		p.Description = description
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockPermRepo) Delete(ctx context.Context, id string) error {
	if p, ok := r.perms[id]; ok {
		delete(r.byName, p.Name)
		delete(r.perms, id)
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockPermRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.perms)), nil
}

func TestRBACService_Roles(t *testing.T) {
	roleRepo := newMockRoleRepo()
	permRepo := newMockPermRepo()
	svc := NewRBACService(roleRepo, permRepo)
	ctx := context.Background()

	// 1. Create Role
	role, err := svc.CreateRole(ctx, models.CreateRoleInput{
		Name:        "editor",
		DisplayName: "Content Editor",
	})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	if role.Name != "editor" || role.DisplayName != "Content Editor" {
		t.Errorf("unexpected role: %+v", role)
	}

	// 2. Duplicate Role Name
	_, err = svc.CreateRole(ctx, models.CreateRoleInput{
		Name:        "editor",
		DisplayName: "Duplicate Editor",
	})
	if !errors.Is(err, ErrRoleNameExists) {
		t.Errorf("expected ErrRoleNameExists, got: %v", err)
	}

	// 3. List Roles
	list, err := svc.ListRoles(ctx)
	if err != nil || len(list) != 1 {
		t.Errorf("expected 1 role, got %d (err: %v)", len(list), err)
	}

	// 4. Update Role
	desc := "Updated description"
	updated, err := svc.UpdateRole(ctx, role.ID, models.UpdateRoleInput{
		DisplayName: "Senior Editor",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("failed to update role: %v", err)
	}
	if updated.DisplayName != "Senior Editor" {
		t.Errorf("expected updated display name 'Senior Editor', got '%s'", updated.DisplayName)
	}

	// 5. Delete System Role Protection
	superAdminRole, _ := svc.CreateRole(ctx, models.CreateRoleInput{
		Name:        models.RoleSuperAdmin,
		DisplayName: "Super Admin",
	})
	err = svc.DeleteRole(ctx, superAdminRole.ID)
	if !errors.Is(err, ErrCannotDeleteSystemRole) {
		t.Errorf("expected ErrCannotDeleteSystemRole, got: %v", err)
	}

	// 6. Delete Custom Role
	err = svc.DeleteRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("failed to delete custom role: %v", err)
	}
}

func TestRBACService_Permissions(t *testing.T) {
	roleRepo := newMockRoleRepo()
	permRepo := newMockPermRepo()
	svc := NewRBACService(roleRepo, permRepo)
	ctx := context.Background()

	// 1. Create Permission
	perm, err := svc.CreatePermission(ctx, models.CreatePermissionInput{
		Name: "articles.publish",
	})
	if err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}
	if perm.Name != "articles.publish" {
		t.Errorf("unexpected permission name: %s", perm.Name)
	}

	// 2. Duplicate Permission
	_, err = svc.CreatePermission(ctx, models.CreatePermissionInput{
		Name: "articles.publish",
	})
	if !errors.Is(err, ErrPermissionNameExists) {
		t.Errorf("expected ErrPermissionNameExists, got: %v", err)
	}

	// 3. Invalid format
	_, err = svc.CreatePermission(ctx, models.CreatePermissionInput{
		Name: "invalid-format-without-dot",
	})
	if err == nil {
		t.Errorf("expected error for invalid permission pattern")
	}

	// 4. List Permissions
	list, err := svc.ListPermissions(ctx)
	if err != nil || len(list) != 1 {
		t.Errorf("expected 1 permission, got %d", len(list))
	}
}
