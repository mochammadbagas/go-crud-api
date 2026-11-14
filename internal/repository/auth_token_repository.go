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

type AuthTokenRepository interface {
	Create(ctx context.Context, token *models.AuthToken) error
	GetByHashAndType(ctx context.Context, tokenHash, tokenType string) (*models.AuthToken, error)
	MarkAsUsed(ctx context.Context, id string) error
	InvalidateUserTokensByType(ctx context.Context, userID, tokenType string) error
	DeleteExpiredOrUsed(ctx context.Context) error
}

type sqlAuthTokenRepository struct {
	q *sqlc.Queries
}

func NewAuthTokenRepository(sqlDB *sql.DB) AuthTokenRepository {
	return &sqlAuthTokenRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlAuthTokenRepository) Create(ctx context.Context, token *models.AuthToken) error {
	now := time.Now().UTC()
	if token.ID == "" {
		token.ID = utils.NewULID()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}

	var payload sql.NullString
	if token.Payload != nil && *token.Payload != "" {
		payload = sql.NullString{String: *token.Payload, Valid: true}
	}

	return r.q.CreateAuthToken(ctx, sqlc.CreateAuthTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		TokenType: sqlc.AuthTokensTokenType(token.TokenType),
		Payload:   payload,
		IsUsed:    token.IsUsed,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	})
}

func (r *sqlAuthTokenRepository) GetByHashAndType(ctx context.Context, tokenHash, tokenType string) (*models.AuthToken, error) {
	row, err := r.q.GetAuthTokenByHashAndType(ctx, sqlc.GetAuthTokenByHashAndTypeParams{
		TokenHash: tokenHash,
		TokenType: sqlc.AuthTokensTokenType(tokenType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var payload *string
	if row.Payload.Valid {
		payload = &row.Payload.String
	}

	return &models.AuthToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		TokenType: string(row.TokenType),
		Payload:   payload,
		IsUsed:    row.IsUsed,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *sqlAuthTokenRepository) MarkAsUsed(ctx context.Context, id string) error {
	return r.q.MarkAuthTokenAsUsed(ctx, id)
}

func (r *sqlAuthTokenRepository) InvalidateUserTokensByType(ctx context.Context, userID, tokenType string) error {
	return r.q.InvalidateUserTokensByType(ctx, sqlc.InvalidateUserTokensByTypeParams{
		UserID:    userID,
		TokenType: sqlc.AuthTokensTokenType(tokenType),
	})
}

func (r *sqlAuthTokenRepository) DeleteExpiredOrUsed(ctx context.Context) error {
	return r.q.DeleteExpiredOrUsedAuthTokens(ctx, time.Now().UTC())
}
