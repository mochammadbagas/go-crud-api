package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate entry")
)

// IsDuplicateKeyError checks if the given error is a MySQL duplicate entry error (code 1062).
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uint64) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Count(ctx context.Context) (int64, error)
}

type sqlUserRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(sqlDB *sql.DB) UserRepository {
	return &sqlUserRepository{
		q: sqlc.New(sqlDB),
	}
}

func (r *sqlUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &models.User{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *sqlUserRepository) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &models.User{
		ID:           row.ID,
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *sqlUserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	res, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	})
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = uint64(id)
	return nil
}

func (r *sqlUserRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}
