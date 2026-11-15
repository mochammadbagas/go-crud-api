package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

func TestAdminUserService_Safeguards(t *testing.T) {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	roleRepo := newMockRoleRepo()
	svc := NewAdminUserService(userRepo, profileRepo, roleRepo)
	ctx := context.Background()

	// 1. Seed Roles
	superAdminRole := &models.Role{
		ID:          utils.NewULID(),
		Name:        models.RoleSuperAdmin,
		DisplayName: "Super Admin",
	}
	_ = roleRepo.Create(ctx, superAdminRole)

	userRole := &models.Role{
		ID:          utils.NewULID(),
		Name:        models.RoleUser,
		DisplayName: "User",
	}
	_ = roleRepo.Create(ctx, userRole)

	// 2. Seed Single Active Super Admin
	superAdminID := utils.NewULID()
	superAdmin := &models.User{
		ID:        superAdminID,
		Username:  "superadmin",
		Email:     "superadmin@example.com",
		Role:      models.RoleSuperAdmin,
		Status:    models.StatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = userRepo.Create(ctx, superAdmin)
	_ = userRepo.AssignRole(ctx, superAdminID, superAdminRole.ID)

	// --- Safeguard 1: Deactivate Last Active Super Admin ---
	_, err := svc.UpdateUserStatus(ctx, superAdminID, models.UpdateUserStatusInput{
		Status: models.StatusInactive,
	})
	if !errors.Is(err, ErrCannotDeactivateLastSuperAdmin) {
		t.Fatalf("expected ErrCannotDeactivateLastSuperAdmin, got: %v", err)
	}

	// --- Safeguard 2: Suspend Last Active Super Admin ---
	_, err = svc.UpdateUserStatus(ctx, superAdminID, models.UpdateUserStatusInput{
		Status: models.StatusSuspended,
	})
	if !errors.Is(err, ErrCannotDeactivateLastSuperAdmin) {
		t.Fatalf("expected ErrCannotDeactivateLastSuperAdmin, got: %v", err)
	}

	// --- Safeguard 3: Delete Last Active Super Admin ---
	err = svc.DeleteUser(ctx, superAdminID)
	if !errors.Is(err, ErrCannotDeleteLastSuperAdmin) {
		t.Fatalf("expected ErrCannotDeleteLastSuperAdmin, got: %v", err)
	}

	// --- Safeguard 4: Demote / Remove super_admin from Last Active Super Admin ---
	_, err = svc.AssignUserRoles(ctx, superAdminID, models.AssignRolesInput{
		RoleIDs: []string{userRole.ID},
	})
	if !errors.Is(err, ErrCannotDemoteLastSuperAdmin) {
		t.Fatalf("expected ErrCannotDemoteLastSuperAdmin, got: %v", err)
	}

	// --- Now Seed a Second Active Super Admin ---
	superAdmin2ID := utils.NewULID()
	superAdmin2 := &models.User{
		ID:        superAdmin2ID,
		Username:  "superadmin2",
		Email:     "superadmin2@example.com",
		Role:      models.RoleSuperAdmin,
		Status:    models.StatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = userRepo.Create(ctx, superAdmin2)
	_ = userRepo.AssignRole(ctx, superAdmin2ID, superAdminRole.ID)

	// Now deactivating the first superadmin should SUCCEED because superadmin2 is still active
	updated, err := svc.UpdateUserStatus(ctx, superAdminID, models.UpdateUserStatusInput{
		Status: models.StatusInactive,
	})
	if err != nil {
		t.Fatalf("expected success with multiple super admins, got: %v", err)
	}
	if updated.Status != models.StatusInactive {
		t.Errorf("expected status inactive, got: %s", updated.Status)
	}

	// Now superAdmin2 is the ONLY ACTIVE super admin!
	// Attempting to delete superAdmin2 must FAIL
	err = svc.DeleteUser(ctx, superAdmin2ID)
	if !errors.Is(err, ErrCannotDeleteLastSuperAdmin) {
		t.Fatalf("expected ErrCannotDeleteLastSuperAdmin on remaining active superadmin, got: %v", err)
	}
}

func TestAdminUserService_CRUD(t *testing.T) {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	roleRepo := newMockRoleRepo()
	svc := NewAdminUserService(userRepo, profileRepo, roleRepo)
	ctx := context.Background()

	// Seed user
	userID := utils.NewULID()
	user := &models.User{
		ID:        userID,
		Username:  "johndoe",
		Email:     "john@example.com",
		Role:      models.RoleUser,
		Status:    models.StatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = userRepo.Create(ctx, user)
	_ = profileRepo.Create(ctx, &models.UserProfile{
		UserID:   userID,
		FullName: "John Doe",
	})

	// 1. Get Detail
	detail, err := svc.GetUserDetail(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get user detail: %v", err)
	}
	if detail.Username != "johndoe" || detail.FullName != "John Doe" {
		t.Errorf("unexpected detail: %+v", detail)
	}

	// 2. Update User by Admin
	updated, err := svc.UpdateUser(ctx, userID, models.AdminUpdateUserInput{
		Username: "johnny",
		Email:    "johnny@example.com",
	})
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}
	if updated.Username != "johnny" || updated.Email != "johnny@example.com" {
		t.Errorf("unexpected updated user: %+v", updated)
	}

	// 3. Delete normal user
	err = svc.DeleteUser(ctx, userID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}
}
