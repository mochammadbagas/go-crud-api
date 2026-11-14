package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrEmailExists        = errors.New("email sudah terdaftar, silakan gunakan email lain")
	ErrUsernameExists     = errors.New("username sudah digunakan, silakan pilih username lain")
	ErrUserNotActive      = errors.New("akun tidak aktif atau sedang ditangguhkan")
	ErrTokenReused        = errors.New("indikasi pembajakan sesi terdeteksi, silakan login ulang")
	ErrTokenExpired       = errors.New("refresh token telah kedaluwarsa")
)

type AuthServiceInterface interface {
	Register(ctx context.Context, input models.RegisterInput) (*models.User, error)
	Login(ctx context.Context, input models.LoginInput, userAgent, ipAddress string) (*models.TokenResponse, error)
	RefreshToken(ctx context.Context, rawRefreshToken, userAgent, ipAddress string) (*models.TokenResponse, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	ChangePassword(ctx context.Context, userID string, input models.ChangePasswordInput) error
}

type AuthService struct {
	userRepo         repository.UserRepository
	profileRepo      repository.UserProfileRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cfg              *config.Config
}

func NewAuthService(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		refreshTokenRepo: refreshTokenRepo,
		cfg:              cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, input models.RegisterInput) (*models.User, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// 1. Check email duplication
	existingUser, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 2. Check username duplication
	existingUsername, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err == nil && existingUsername != nil {
		return nil, ErrUsernameExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 3. Hash password with Argon2id
	passwordHash, err := argon2id.CreateHash(input.Password, argon2id.DefaultParams)
	if err != nil {
		return nil, err
	}

	// 4. Create User entity
	user := &models.User{
		ID:           utils.NewULID(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
		Role:         models.RoleUser,
		Status:       models.StatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrEmailExists
		}
		return nil, err
	}

	// 5. Create default user profile
	fullName := input.FullName
	if fullName == "" {
		fullName = input.Username
	}
	_ = s.profileRepo.Create(ctx, &models.UserProfile{
		UserID:    user.ID,
		FullName:  fullName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input models.LoginInput, userAgent, ipAddress string) (*models.TokenResponse, error) {
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

	// Verify Argon2id password hash
	match, err := argon2id.ComparePasswordAndHash(input.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}

	// Check user status
	if user.Status != models.StatusActive {
		return nil, ErrUserNotActive
	}

	// Generate JWT Access Token
	accessToken, err := utils.GenerateAccessToken(
		s.cfg.JWTSecret,
		s.cfg.JWTAccessTTL,
		user.ID,
		user.Username,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, err
	}

	// Generate Refresh Token & Token Family
	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	tokenHash := utils.HashToken(rawRefreshToken)
	tokenFamily := utils.GenerateTokenFamily()
	expiresAt := time.Now().UTC().Add(s.cfg.JWTRefreshTTL)

	var uaPtr, ipPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}
	if ipAddress != "" {
		ipPtr = &ipAddress
	}

	refreshTokenModel := &models.RefreshToken{
		ID:          utils.NewULID(),
		UserID:      user.ID,
		TokenHash:   tokenHash,
		TokenFamily: tokenFamily,
		IsRevoked:   false,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
		UserAgent:   uaPtr,
		IPAddress:   ipPtr,
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshTokenModel); err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken, userAgent, ipAddress string) (*models.TokenResponse, error) {
	if rawRefreshToken == "" {
		return nil, ErrUnauthorized
	}

	tokenHash := utils.HashToken(rawRefreshToken)
	storedToken, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	// REUSE DETECTION: If token was already revoked, revoke the entire token family!
	if storedToken.IsRevoked {
		slog.Warn("Deteksi penggunaan ulang refresh token! Mencabut seluruh token family",
			slog.String("family", storedToken.TokenFamily),
			slog.String("user_id", storedToken.UserID),
		)
		_ = s.refreshTokenRepo.RevokeFamily(ctx, storedToken.TokenFamily)
		return nil, ErrTokenReused
	}

	// Check token expiration
	if time.Now().UTC().After(storedToken.ExpiresAt) {
		_ = s.refreshTokenRepo.Revoke(ctx, storedToken.ID)
		return nil, ErrTokenExpired
	}

	// Fetch user & verify status
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if user.Status != models.StatusActive {
		return nil, ErrUserNotActive
	}

	// 1. Revoke the old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, storedToken.ID); err != nil {
		return nil, err
	}

	// 2. Generate a new rotated refresh token in the same token family
	newRawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	newTokenHash := utils.HashToken(newRawRefreshToken)
	newExpiresAt := time.Now().UTC().Add(s.cfg.JWTRefreshTTL)

	var uaPtr, ipPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}
	if ipAddress != "" {
		ipPtr = &ipAddress
	}

	newRefreshTokenModel := &models.RefreshToken{
		ID:          utils.NewULID(),
		UserID:      user.ID,
		TokenHash:   newTokenHash,
		TokenFamily: storedToken.TokenFamily,
		IsRevoked:   false,
		ExpiresAt:   newExpiresAt,
		CreatedAt:   time.Now().UTC(),
		UserAgent:   uaPtr,
		IPAddress:   ipPtr,
	}

	if err := s.refreshTokenRepo.Create(ctx, newRefreshTokenModel); err != nil {
		return nil, err
	}

	// 3. Generate a new Access Token
	accessToken, err := utils.GenerateAccessToken(
		s.cfg.JWTSecret,
		s.cfg.JWTAccessTTL,
		user.ID,
		user.Username,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, err
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	tokenHash := utils.HashToken(rawRefreshToken)
	storedToken, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	return s.refreshTokenRepo.Revoke(ctx, storedToken.ID)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID string, input models.ChangePasswordInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Validate old password
	match, err := argon2id.ComparePasswordAndHash(input.OldPassword, user.PasswordHash)
	if err != nil || !match {
		return errors.New("password lama salah")
	}

	// Hash new password
	newHash, err := argon2id.CreateHash(input.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	// Update password in DB
	if err := s.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}

	// Revoke all active refresh tokens for this user
	return s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
}
