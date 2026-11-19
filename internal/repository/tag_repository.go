package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	ErrTagNotFound   = errors.New("tag not found")
	ErrTagSlugExists = errors.New("tag slug is already in use")
)

type TagRepository interface {
	List(ctx context.Context) ([]models.Tag, error)
	GetByID(ctx context.Context, id string) (*models.Tag, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tag, error)
	GetByName(ctx context.Context, name string) (*models.Tag, error)
	Create(ctx context.Context, tag *models.Tag) error
	Update(ctx context.Context, tag *models.Tag) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
	GetOrCreateTags(ctx context.Context, tagNames []string) ([]models.Tag, error)
}

type sqlTagRepository struct {
	q *sqlc.Queries
}

func NewTagRepository(sqlDB *sql.DB) TagRepository {
	return &sqlTagRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlTagRepository) List(ctx context.Context) ([]models.Tag, error) {
	rows, err := r.q.ListTags(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Tag, len(rows))
	for i, row := range rows {
		result[i] = models.Tag{
			ID:         row.ID,
			Name:       row.Name,
			Slug:       row.Slug,
			UsageCount: row.UsageCount,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
	}
	return result, nil
}

func (r *sqlTagRepository) GetByID(ctx context.Context, id string) (*models.Tag, error) {
	row, err := r.q.GetTagByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	return &models.Tag{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlTagRepository) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	row, err := r.q.GetTagBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	return &models.Tag{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlTagRepository) GetByName(ctx context.Context, name string) (*models.Tag, error) {
	row, err := r.q.GetTagByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	return &models.Tag{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlTagRepository) Create(ctx context.Context, tag *models.Tag) error {
	if tag.ID == "" {
		tag.ID = utils.NewULID()
	}
	now := time.Now().UTC()
	tag.CreatedAt = now
	tag.UpdatedAt = now

	return r.q.CreateTag(ctx, sqlc.CreateTagParams{
		ID:        tag.ID,
		Name:      tag.Name,
		Slug:      tag.Slug,
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	})
}

func (r *sqlTagRepository) Update(ctx context.Context, tag *models.Tag) error {
	tag.UpdatedAt = time.Now().UTC()

	return r.q.UpdateTag(ctx, sqlc.UpdateTagParams{
		ID:        tag.ID,
		Name:      tag.Name,
		Slug:      tag.Slug,
		UpdatedAt: tag.UpdatedAt,
	})
}

func (r *sqlTagRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteTag(ctx, id)
}

func (r *sqlTagRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountTags(ctx)
}

func (r *sqlTagRepository) GetOrCreateTags(ctx context.Context, tagNames []string) ([]models.Tag, error) {
	if len(tagNames) == 0 {
		return []models.Tag{}, nil
	}

	seen := make(map[string]bool)
	var result []models.Tag

	for _, rawName := range tagNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		slug := utils.GenerateSlug(name)
		if seen[slug] {
			continue
		}
		seen[slug] = true

		tag, err := r.GetBySlug(ctx, slug)
		if err == nil && tag != nil {
			result = append(result, *tag)
			continue
		}

		// Create tag on the fly
		newTag := &models.Tag{
			ID:   utils.NewULID(),
			Name: name,
			Slug: slug,
		}
		if err := r.Create(ctx, newTag); err != nil {
			// In case of parallel race condition
			existing, getErr := r.GetBySlug(ctx, slug)
			if getErr == nil && existing != nil {
				result = append(result, *existing)
				continue
			}
			return nil, err
		}
		result = append(result, *newTag)
	}

	return result, nil
}
