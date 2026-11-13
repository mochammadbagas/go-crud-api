package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
)

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (*models.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}

type sqlSessionRepository struct {
	q *sqlc.Queries
}

func NewSessionRepository(sqlDB *sql.DB) SessionRepository {
	return &sqlSessionRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlSessionRepository) Create(ctx context.Context, session *models.Session) error {
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}

	return r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        session.ID,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	})
}

func (r *sqlSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	now := time.Now().UTC()
	row, err := r.q.GetSessionByID(ctx, sqlc.GetSessionByIDParams{
		ID:        id,
		ExpiresAt: now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &models.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
		User: &models.User{
			ID:    row.UserID,
			Name:  row.Name,
			Email: row.Email,
		},
	}, nil
}

func (r *sqlSessionRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteSession(ctx, id)
}

func (r *sqlSessionRepository) DeleteExpired(ctx context.Context) error {
	return r.q.DeleteExpiredSessions(ctx, time.Now().UTC())
}
