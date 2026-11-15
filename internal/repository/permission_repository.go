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

type PermissionRepository interface {
	Create(ctx context.Context, perm *models.Permission) error
	GetByID(ctx context.Context, id string) (*models.Permission, error)
	GetByName(ctx context.Context, name string) (*models.Permission, error)
	List(ctx context.Context) ([]*models.Permission, error)
	Update(ctx context.Context, id string, description *string) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type sqlPermissionRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return &sqlPermissionRepository{
		db: db,
		q:  sqlc.New(db),
	}
}

func (r *sqlPermissionRepository) Create(ctx context.Context, perm *models.Permission) error {
	if perm.ID == "" {
		perm.ID = utils.NewULID()
	}
	now := time.Now().UTC()
	if perm.CreatedAt.IsZero() {
		perm.CreatedAt = now
	}
	perm.UpdatedAt = now

	var desc sql.NullString
	if perm.Description != nil {
		desc = sql.NullString{String: *perm.Description, Valid: true}
	}

	err := r.q.CreatePermission(ctx, sqlc.CreatePermissionParams{
		ID:          perm.ID,
		Name:        perm.Name,
		Description: desc,
		CreatedAt:   perm.CreatedAt,
		UpdatedAt:   perm.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlPermissionRepository) GetByID(ctx context.Context, id string) (*models.Permission, error) {
	row, err := r.q.GetPermissionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	perm := &models.Permission{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Description.Valid {
		perm.Description = &row.Description.String
	}
	return perm, nil
}

func (r *sqlPermissionRepository) GetByName(ctx context.Context, name string) (*models.Permission, error) {
	row, err := r.q.GetPermissionByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	perm := &models.Permission{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Description.Valid {
		perm.Description = &row.Description.String
	}
	return perm, nil
}

func (r *sqlPermissionRepository) List(ctx context.Context) ([]*models.Permission, error) {
	rows, err := r.q.ListPermissions(ctx)
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

func (r *sqlPermissionRepository) Update(ctx context.Context, id string, description *string) error {
	var desc sql.NullString
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}

	return r.q.UpdatePermission(ctx, sqlc.UpdatePermissionParams{
		ID:          id,
		Description: desc,
		UpdatedAt:   time.Now().UTC(),
	})
}

func (r *sqlPermissionRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeletePermission(ctx, id)
}

func (r *sqlPermissionRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountPermissions(ctx)
}
