package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"

	"github.com/alexedwards/argon2id"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUnauthorized       = errors.New("unauthorized")
)

// AuthServiceInterface defines business actions for authentication and session management.
type AuthServiceInterface interface {
	Login(ctx context.Context, input models.LoginInput) (*models.Session, error)
	Logout(ctx context.Context, sessionID string) error
	ValidateSession(ctx context.Context, sessionID string) (*models.Session, error)
}

type AuthService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewAuthService(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Login authenticates the administrator and returns an active Session.
func (s *AuthService) Login(ctx context.Context, input models.LoginInput) (*models.Session, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Password verification with Argon2id
	match, err := argon2id.ComparePasswordAndHash(input.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}

	// Generate 32 bytes cryptographically secure random session ID
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return nil, err
	}
	sessionID := hex.EncodeToString(sessionBytes)

	// Session valid for 7 days
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	session := &models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
		User:      user,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// Logout invalidates the session in the database.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessionRepo.Delete(ctx, sessionID)
}

// ValidateSession verifies if the session ID is valid and active.
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*models.Session, error) {
	if sessionID == "" {
		return nil, ErrUnauthorized
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	// Defense-in-depth: While the repository query filters out expired records at the database level,
	// checking IsExpired() here guards against clock drift between the application server and MySQL server.
	if session.IsExpired() {
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, ErrUnauthorized
	}

	return session, nil
}

// HashPassword hashes a plain-text password using Argon2id.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

