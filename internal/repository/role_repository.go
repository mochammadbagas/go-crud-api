package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id string) (*models.Role, error)
	GetByName(ctx context.Context, name string) (*models.Role, error)
	List(ctx context.Context) ([]*models.Role, error)
	Update(ctx context.Context, id string, displayName string, description *string) error
	Delete(ctx context.Context, id string) error
	AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*models.Permission, error)
	Count(ctx context.Context) (int64, error)
}

type sqlRoleRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewRoleRepository(db *sql.DB) RoleRepository {
	return &sqlRoleRepository{
		db: db,
		q:  sqlc.New(db),
	}
}

func (r *sqlRoleRepository) Create(ctx context.Context, role *models.Role) error {
	if role.ID == "" {
		role.ID = utils.NewULID()
	}
	now := time.Now().UTC()
	if role.CreatedAt.IsZero() {
		role.CreatedAt = now
	}
	role.UpdatedAt = now

	var desc sql.NullString
	if role.Description != nil {
		desc = sql.NullString{String: *role.Description, Valid: true}
	}

	err := r.q.CreateRole(ctx, sqlc.CreateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: desc,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}

	if len(role.Permissions) > 0 {
		return r.AssignPermissions(ctx, role.ID, role.Permissions)
	}

	return nil
}

func (r *sqlRoleRepository) GetByID(ctx context.Context, id string) (*models.Role, error) {
	row, err := r.q.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	role := &models.Role{
		ID:          row.ID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Description.Valid {
		role.Description = &row.Description.String
	}

	perms, err := r.GetPermissionsByRoleID(ctx, role.ID)
	if err == nil {
		permNames := make([]string, len(perms))
		for i, p := range perms {
			permNames[i] = p.Name
		}
		role.Permissions = permNames
	}

	return role, nil
}

func (r *sqlRoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	row, err := r.q.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	role := &models.Role{
		ID:          row.ID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Description.Valid {
		role.Description = &row.Description.String
	}

	return role, nil
}

func (r *sqlRoleRepository) List(ctx context.Context) ([]*models.Role, error) {
	rows, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*models.Role, len(rows))
	for i, row := range rows {
		role := &models.Role{
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: row.DisplayName,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
		if row.Description.Valid {
			role.Description = &row.Description.String
		}
		result[i] = role
	}

	return result, nil
}

func (r *sqlRoleRepository) Update(ctx context.Context, id string, displayName string, description *string) error {
	var desc sql.NullString
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}

	return r.q.UpdateRole(ctx, sqlc.UpdateRoleParams{
		DisplayName: displayName,
		Description: desc,
		UpdatedAt:   time.Now().UTC(),
		ID:          id,
	})
}

func (r *sqlRoleRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteRole(ctx, id)
}

func (r *sqlRoleRepository) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	// Remove existing permissions
	if err := qtx.RemoveAllPermissionsFromRole(ctx, roleID); err != nil {
		return err
	}

	permissionIDs = utils.DeduplicateStrings(permissionIDs)
	now := time.Now().UTC()
	for _, permID := range permissionIDs {
		if permID == "" {
			continue
		}
		err := qtx.AssignPermissionToRole(ctx, sqlc.AssignPermissionToRoleParams{
			RoleID:       roleID,
			PermissionID: permID,
			CreatedAt:    now,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *sqlRoleRepository) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*models.Permission, error) {
	rows, err := r.q.GetPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	result := make([]*models.Permission, len(rows))
	for i, row := range rows {
		p := &models.Permission{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.Description.Valid {
			p.Description = &row.Description.String
		}
		result[i] = p
	}

	return result, nil
}

func (r *sqlRoleRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountRoles(ctx)
}
