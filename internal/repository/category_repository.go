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

var (
	ErrCategoryHasArticles = errors.New("category cannot be deleted because it is still used by one or more articles")
)

type CategoryRepository interface {
	List(ctx context.Context) ([]models.Category, error)
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Create(ctx context.Context, cat *models.Category) error
	Update(ctx context.Context, cat *models.Category) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
	CountArticlesByCategoryID(ctx context.Context, categoryID string) (int64, error)
}

type sqlCategoryRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewCategoryRepository(sqlDB *sql.DB) CategoryRepository {
	return &sqlCategoryRepository{
		db: sqlDB,
		q:  sqlc.New(sqlDB),
	}
}

func (r *sqlCategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Category, 0, len(rows))
	for _, row := range rows {
		c := models.Category{
			ID:           row.ID,
			Name:         row.Name,
			Slug:         row.Slug,
			ArticleCount: row.ArticleCount,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
		if row.Description.Valid {
			desc := row.Description.String
			c.Description = &desc
		}
		result = append(result, c)
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

	c := &models.Category{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Description.Valid {
		desc := row.Description.String
		c.Description = &desc
	}
	return c, nil
}

func (r *sqlCategoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	row, err := r.q.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	c := &models.Category{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.Description.Valid {
		desc := row.Description.String
		c.Description = &desc
	}
	return c, nil
}

func (r *sqlCategoryRepository) Create(ctx context.Context, cat *models.Category) error {
	if cat.ID == "" {
		cat.ID = utils.NewULID()
	}
	now := time.Now().UTC()
	cat.CreatedAt = now
	cat.UpdatedAt = now

	params := sqlc.CreateCategoryParams{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		CreatedAt: cat.CreatedAt,
		UpdatedAt: cat.UpdatedAt,
	}
	if cat.Description != nil {
		params.Description = sql.NullString{String: *cat.Description, Valid: true}
	}

	return r.q.CreateCategory(ctx, params)
}

func (r *sqlCategoryRepository) Update(ctx context.Context, cat *models.Category) error {
	cat.UpdatedAt = time.Now().UTC()

	params := sqlc.UpdateCategoryParams{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		UpdatedAt: cat.UpdatedAt,
	}
	if cat.Description != nil {
		params.Description = sql.NullString{String: *cat.Description, Valid: true}
	}

	return r.q.UpdateCategory(ctx, params)
}

func (r *sqlCategoryRepository) Delete(ctx context.Context, id string) error {
	if r.db != nil {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		qtx := r.q.WithTx(tx)
		// Detach category from any soft-deleted articles so foreign key constraints are not violated
		if err := qtx.DetachCategoryFromSoftDeletedArticles(ctx, sql.NullString{String: id, Valid: true}); err != nil {
			return err
		}
		if err := qtx.DeleteCategory(ctx, id); err != nil {
			return err
		}
		return tx.Commit()
	}

	_ = r.q.DetachCategoryFromSoftDeletedArticles(ctx, sql.NullString{String: id, Valid: true})
	return r.q.DeleteCategory(ctx, id)
}

func (r *sqlCategoryRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountCategories(ctx)
}

func (r *sqlCategoryRepository) CountArticlesByCategoryID(ctx context.Context, categoryID string) (int64, error) {
	return r.q.CountArticlesByCategoryID(ctx, sql.NullString{String: categoryID, Valid: true})
}
