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

type CategoryRepository interface {
	List(ctx context.Context) ([]models.Category, error)
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Create(ctx context.Context, cat *models.Category) error
	Update(ctx context.Context, cat *models.Category) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type sqlCategoryRepository struct {
	q *sqlc.Queries
}

func NewCategoryRepository(sqlDB *sql.DB) CategoryRepository {
	return &sqlCategoryRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Category, 0, len(rows))
	for _, row := range rows {
		result = append(result, models.Category{
			ID:        row.ID,
			Name:      row.Name,
			Slug:      row.Slug,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return result, nil
}

func (r *sqlCategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	row, err := r.q.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &models.Category{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlCategoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	row, err := r.q.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &models.Category{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlCategoryRepository) Create(ctx context.Context, cat *models.Category) error {
	now := time.Now().UTC()
	if cat.ID == "" {
		cat.ID = utils.NewULID()
	}
	if cat.CreatedAt.IsZero() {
		cat.CreatedAt = now
	}
	if cat.UpdatedAt.IsZero() {
		cat.UpdatedAt = now
	}

	err := r.q.CreateCategory(ctx, sqlc.CreateCategoryParams{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}

	return nil
}

func (r *sqlCategoryRepository) Update(ctx context.Context, cat *models.Category) error {
	cat.UpdatedAt = time.Now().UTC()
	err := r.q.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name:      cat.Name,
		Slug:      cat.Slug,
		UpdatedAt: cat.UpdatedAt,
		ID:        cat.ID,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlCategoryRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteCategory(ctx, id)
}

func (r *sqlCategoryRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountCategories(ctx)
}
