package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
)

type UserProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.UserProfile, error)
	Create(ctx context.Context, profile *models.UserProfile) error
	Update(ctx context.Context, profile *models.UserProfile) error
	UpdateAvatar(ctx context.Context, userID string, avatarURL string) error
	DeleteAvatar(ctx context.Context, userID string) error
	GetUserWithProfile(ctx context.Context, userID string) (*models.UserWithProfile, error)
}

type sqlUserProfileRepository struct {
	q *sqlc.Queries
}

func NewUserProfileRepository(sqlDB *sql.DB) UserProfileRepository {
	return &sqlUserProfileRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlUserProfileRepository) GetByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	row, err := r.q.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var bio, phone, avatar *string
	if row.Bio.Valid {
		bio = &row.Bio.String
	}
	if row.PhoneNumber.Valid {
		phone = &row.PhoneNumber.String
	}
	if row.AvatarUrl.Valid {
		avatar = &row.AvatarUrl.String
	}

	return &models.UserProfile{
		UserID:      row.UserID,
		FullName:    row.FullName,
		Bio:         bio,
		PhoneNumber: phone,
		AvatarURL:   avatar,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *sqlUserProfileRepository) Create(ctx context.Context, profile *models.UserProfile) error {
	now := time.Now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = now
	}

	var bio, phone, avatar sql.NullString
	if profile.Bio != nil {
		bio = sql.NullString{String: *profile.Bio, Valid: true}
	}
	if profile.PhoneNumber != nil {
		phone = sql.NullString{String: *profile.PhoneNumber, Valid: true}
	}
	if profile.AvatarURL != nil {
		avatar = sql.NullString{String: *profile.AvatarURL, Valid: true}
	}

	return r.q.CreateUserProfile(ctx, sqlc.CreateUserProfileParams{
		UserID:      profile.UserID,
		FullName:    profile.FullName,
		Bio:         bio,
		PhoneNumber: phone,
		AvatarUrl:   avatar,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
	})
}

func (r *sqlUserProfileRepository) Update(ctx context.Context, profile *models.UserProfile) error {
	var bio, phone sql.NullString
	if profile.Bio != nil {
		bio = sql.NullString{String: *profile.Bio, Valid: true}
	}
	if profile.PhoneNumber != nil {
		phone = sql.NullString{String: *profile.PhoneNumber, Valid: true}
	}

	return r.q.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		FullName:    profile.FullName,
		Bio:         bio,
		PhoneNumber: phone,
		UpdatedAt:   time.Now().UTC(),
		UserID:      profile.UserID,
	})
}

func (r *sqlUserProfileRepository) UpdateAvatar(ctx context.Context, userID string, avatarURL string) error {
	return r.q.UpdateAvatar(ctx, sqlc.UpdateAvatarParams{
		AvatarUrl: sql.NullString{String: avatarURL, Valid: true},
		UpdatedAt: time.Now().UTC(),
		UserID:    userID,
	})
}

func (r *sqlUserProfileRepository) DeleteAvatar(ctx context.Context, userID string) error {
	return r.q.DeleteAvatar(ctx, sqlc.DeleteAvatarParams{
		UpdatedAt: time.Now().UTC(),
		UserID:    userID,
	})
}

func (r *sqlUserProfileRepository) GetUserWithProfile(ctx context.Context, userID string) (*models.UserWithProfile, error) {
	row, err := r.q.GetUserWithProfileByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var verifiedAt *time.Time
	if row.EmailVerifiedAt.Valid {
		t := row.EmailVerifiedAt.Time
		verifiedAt = &t
	}

	var bio, phone, avatar *string
	if row.Bio.Valid {
		bio = &row.Bio.String
	}
	if row.PhoneNumber.Valid {
		phone = &row.PhoneNumber.String
	}
	if row.AvatarUrl.Valid {
		avatar = &row.AvatarUrl.String
	}

	fullName := ""
	if row.FullName.Valid {
		fullName = row.FullName.String
	}

	return &models.UserWithProfile{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		FullName:        fullName,
		Bio:             bio,
		PhoneNumber:     phone,
		AvatarURL:       avatar,
	}, nil
}
