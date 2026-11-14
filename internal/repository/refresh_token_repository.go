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

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeFamily(ctx context.Context, tokenFamily string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

type sqlRefreshTokenRepository struct {
	q *sqlc.Queries
}

func NewRefreshTokenRepository(sqlDB *sql.DB) RefreshTokenRepository {
	return &sqlRefreshTokenRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlRefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	now := time.Now().UTC()
	if token.ID == "" {
		token.ID = utils.NewULID()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}

	var userAgent, ipAddress sql.NullString
	if token.UserAgent != nil {
		userAgent = sql.NullString{String: *token.UserAgent, Valid: true}
	}
	if token.IPAddress != nil {
		ipAddress = sql.NullString{String: *token.IPAddress, Valid: true}
	}

	return r.q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		ID:          token.ID,
		UserID:      token.UserID,
		TokenHash:   token.TokenHash,
		TokenFamily: token.TokenFamily,
		IsRevoked:   token.IsRevoked,
		ExpiresAt:   token.ExpiresAt,
		CreatedAt:   token.CreatedAt,
		UserAgent:   userAgent,
		IpAddress:   ipAddress,
	})
}

func (r *sqlRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	row, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var userAgent, ipAddress *string
	if row.UserAgent.Valid {
		userAgent = &row.UserAgent.String
	}
	if row.IpAddress.Valid {
		ipAddress = &row.IpAddress.String
	}

	return &models.RefreshToken{
		ID:          row.ID,
		UserID:      row.UserID,
		TokenHash:   row.TokenHash,
		TokenFamily: row.TokenFamily,
		IsRevoked:   row.IsRevoked,
		ExpiresAt:   row.ExpiresAt,
		CreatedAt:   row.CreatedAt,
		UserAgent:   userAgent,
		IPAddress:   ipAddress,
	}, nil
}

func (r *sqlRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	return r.q.RevokeRefreshToken(ctx, id)
}

func (r *sqlRefreshTokenRepository) RevokeFamily(ctx context.Context, tokenFamily string) error {
	return r.q.RevokeTokenFamily(ctx, tokenFamily)
}

func (r *sqlRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	return r.q.RevokeAllUserTokens(ctx, userID)
}

func (r *sqlRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.q.DeleteExpiredRefreshTokens(ctx, time.Now().UTC())
}
