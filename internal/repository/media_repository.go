package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
)

var (
	ErrMediaNotFound = errors.New("media not found")
	ErrMediaInUse    = errors.New("media is currently referenced by one or more articles and cannot be deleted")
)

type MediaRepository interface {
	Create(ctx context.Context, media *models.Media) error
	GetByID(ctx context.Context, id string) (*models.Media, error)
	List(ctx context.Context, limit, offset int32) ([]models.Media, int64, error)
	Delete(ctx context.Context, id string) error
	IsInUse(ctx context.Context, id string) (bool, error)
}

type sqlMediaRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewMediaRepository(db *sql.DB) MediaRepository {
	return &sqlMediaRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *sqlMediaRepository) Create(ctx context.Context, media *models.Media) error {
	params := sqlc.CreateMediaParams{
		ID:          media.ID,
		UploadedBy:  media.UploadedBy,
		Filename:    media.Filename,
		StorageDisk: media.StorageDisk,
		StorageKey:  media.StorageKey,
		Url:         media.URL,
		MimeType:    media.MimeType,
		Size:        media.Size,
		CreatedAt:   media.CreatedAt,
		UpdatedAt:   media.UpdatedAt,
	}
	if media.Width != nil {
		params.Width = sql.NullInt32{Int32: *media.Width, Valid: true}
	}
	if media.Height != nil {
		params.Height = sql.NullInt32{Int32: *media.Height, Valid: true}
	}

	return r.queries.CreateMedia(ctx, params)
}

func (r *sqlMediaRepository) GetByID(ctx context.Context, id string) (*models.Media, error) {
	row, err := r.queries.GetMediaByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMediaNotFound
		}
		return nil, err
	}

	m := &models.Media{
		ID:          row.ID,
		UploadedBy:  row.UploadedBy,
		Filename:    row.Filename,
		StorageDisk: row.StorageDisk,
		StorageKey:  row.StorageKey,
		URL:         row.Url,
		MimeType:    row.MimeType,
		Size:        row.Size,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Width.Valid {
		w := row.Width.Int32
		m.Width = &w
	}
	if row.Height.Valid {
		h := row.Height.Int32
		m.Height = &h
	}

	return m, nil
}

func (r *sqlMediaRepository) List(ctx context.Context, limit, offset int32) ([]models.Media, int64, error) {
	total, err := r.queries.CountMedia(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.queries.ListMedia(ctx, sqlc.ListMediaParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	medias := make([]models.Media, len(rows))
	for i, row := range rows {
		m := models.Media{
			ID:          row.ID,
			UploadedBy:  row.UploadedBy,
			Filename:    row.Filename,
			StorageDisk: row.StorageDisk,
			StorageKey:  row.StorageKey,
			URL:         row.Url,
			MimeType:    row.MimeType,
			Size:        row.Size,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
		if row.Width.Valid {
			w := row.Width.Int32
			m.Width = &w
		}
		if row.Height.Valid {
			h := row.Height.Int32
			m.Height = &h
		}
		medias[i] = m
	}

	return medias, total, nil
}

func (r *sqlMediaRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteMedia(ctx, id)
}

func (r *sqlMediaRepository) IsInUse(ctx context.Context, id string) (bool, error) {
	res, err := r.queries.IsMediaInUse(ctx, sqlc.IsMediaInUseParams{
		FeaturedImageID: sql.NullString{String: id, Valid: true},
		OgImageID:       sql.NullString{String: id, Valid: true},
		MediaID:         id,
	})
	if err != nil {
		return false, err
	}
	return res.Valid && res.Bool, nil
}
