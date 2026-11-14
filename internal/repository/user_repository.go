package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

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
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	UpdateUsername(ctx context.Context, id string, username string) error
	UpdateStatus(ctx context.Context, id string, status string) error
	VerifyEmail(ctx context.Context, id string) error
	UpdateEmail(ctx context.Context, id string, newEmail string) error
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

	var verifiedAt *time.Time
	if row.EmailVerifiedAt.Valid {
		t := row.EmailVerifiedAt.Time
		verifiedAt = &t
	}

	return &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *sqlUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row, err := r.q.GetUserByUsername(ctx, username)
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

	return &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *sqlUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
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

	return &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *sqlUserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = utils.NewULID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	if user.Role == "" {
		user.Role = models.RoleUser
	}
	if user.Status == "" {
		user.Status = models.StatusActive
	}

	var emailVerifiedAt sql.NullTime
	if user.EmailVerifiedAt != nil {
		emailVerifiedAt = sql.NullTime{Time: *user.EmailVerifiedAt, Valid: true}
	}

	err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		PasswordHash:    user.PasswordHash,
		Role:            sqlc.UsersRole(user.Role),
		Status:          sqlc.UsersStatus(user.Status),
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	return r.q.UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    time.Now().UTC(),
		ID:           id,
	})
}

func (r *sqlUserRepository) UpdateUsername(ctx context.Context, id string, username string) error {
	err := r.q.UpdateUsername(ctx, sqlc.UpdateUsernameParams{
		Username:  username,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.q.UpdateStatus(ctx, sqlc.UpdateStatusParams{
		Status:    sqlc.UsersStatus(status),
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
}

func (r *sqlUserRepository) VerifyEmail(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.q.VerifyUserEmail(ctx, sqlc.VerifyUserEmailParams{
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:       now,
		ID:              id,
	})
}

func (r *sqlUserRepository) UpdateEmail(ctx context.Context, id string, newEmail string) error {
	now := time.Now().UTC()
	err := r.q.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{
		Email:           newEmail,
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:       now,
		ID:              id,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}
