package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrEmailExists          = errors.New("email is already registered")
	ErrUsernameExists       = errors.New("username is already taken")
	ErrUserNotActive        = errors.New("account is inactive or suspended")
	ErrTokenReused          = errors.New("session hijacking detected, please log in again")
	ErrTokenExpired         = errors.New("refresh token has expired")
	ErrAuthTokenInvalid     = errors.New("invalid or expired token")
	ErrAuthTokenUsed        = errors.New("token has already been used")
	ErrEmailAlreadyVerified = errors.New("email address is already verified")
)

type AuthServiceInterface interface {
	Register(ctx context.Context, input models.RegisterInput) (*models.User, error)
	Login(ctx context.Context, input models.LoginInput, userAgent, ipAddress string) (*models.TokenResponse, error)
	RefreshToken(ctx context.Context, rawRefreshToken, userAgent, ipAddress string) (*models.TokenResponse, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	ChangePassword(ctx context.Context, userID string, input models.ChangePasswordInput) error
	ForgotPassword(ctx context.Context, input models.ForgotPasswordInput) error
	ResetPassword(ctx context.Context, input models.ResetPasswordInput) error
	VerifyEmail(ctx context.Context, input models.VerifyEmailInput) error
	ResendVerification(ctx context.Context, input models.ResendVerificationInput) error
	SendVerification(ctx context.Context, userID string) error
}

type AuthService struct {
	userRepo         repository.UserRepository
	profileRepo      repository.UserProfileRepository
	refreshTokenRepo repository.RefreshTokenRepository
	authTokenRepo    repository.AuthTokenRepository
	roleRepo         repository.RoleRepository
	mailer           mailer.Mailer
	cfg              *config.Config
}

func NewAuthService(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	authTokenRepo repository.AuthTokenRepository,
	mail mailer.Mailer,
	cfg *config.Config,
	roleRepo ...repository.RoleRepository,
) *AuthService {
	var rRepo repository.RoleRepository
	if len(roleRepo) > 0 {
		rRepo = roleRepo[0]
	}
	return &AuthService{
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		refreshTokenRepo: refreshTokenRepo,
		authTokenRepo:    authTokenRepo,
		roleRepo:         rRepo,
		mailer:           mail,
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

	// 6. Assign default role 'user'
	if s.roleRepo != nil {
		if defaultRole, err := s.roleRepo.GetByName(ctx, models.RoleUser); err == nil && defaultRole != nil {
			_ = s.userRepo.AssignRole(ctx, user.ID, defaultRole.ID)
			user.Roles = []string{defaultRole.Name}
		}
	}
	if len(user.Roles) == 0 {
		user.Roles = []string{user.Role}
	}

	// 7. Send initial email verification in background
	go func() {
		_ = s.SendVerification(context.Background(), user.ID)
	}()

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
		user.Roles...,
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
		slog.Warn("Refresh token reuse detected! Revoking full token family",
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
		user.Roles...,
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
		return errors.New("incorrect old password")
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

func (s *AuthService) ForgotPassword(ctx context.Context, input models.ForgotPasswordInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		// Anti user-enumeration: return success even if user not found
		if errors.Is(err, repository.ErrNotFound) {
			slog.Info("Forgot password diminta untuk email yang tidak terdaftar (anti-enumeration diterapkan)", slog.String("email", input.Email))
			return nil
		}
		return err
	}

	// Invalidate any existing unused reset tokens for this user
	_ = s.authTokenRepo.InvalidateUserTokensByType(ctx, user.ID, models.TokenTypePasswordReset)

	// Generate secure raw token
	rawToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return err
	}

	tokenHash := utils.HashToken(rawToken)
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	tokenModel := &models.AuthToken{
		ID:        utils.NewULID(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		TokenType: models.TokenTypePasswordReset,
		IsUsed:    false,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.authTokenRepo.Create(ctx, tokenModel); err != nil {
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.FrontendURL, rawToken)
	if s.mailer != nil {
		if err := s.mailer.SendResetPasswordEmail(ctx, user.Email, resetLink); err != nil {
			slog.Error("Failed to send password reset email", slog.String("email", user.Email), slog.Any("error", err))
		}
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, input models.ResetPasswordInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	tokenHash := utils.HashToken(input.Token)
	storedToken, err := s.authTokenRepo.GetByHashAndType(ctx, tokenHash, models.TokenTypePasswordReset)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAuthTokenInvalid
		}
		return err
	}

	if storedToken.IsUsed {
		return ErrAuthTokenUsed
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return ErrAuthTokenInvalid
	}

	// Hash new password
	newHash, err := argon2id.CreateHash(input.NewPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	// Update user's password
	if err := s.userRepo.UpdatePassword(ctx, storedToken.UserID, newHash); err != nil {
		return err
	}

	// Mark token as used
	if err := s.authTokenRepo.MarkAsUsed(ctx, storedToken.ID); err != nil {
		slog.Error("Failed to mark auth token as used", slog.String("id", storedToken.ID), slog.Any("error", err))
	}

	// Revoke all active sessions
	return s.refreshTokenRepo.RevokeAllForUser(ctx, storedToken.UserID)
}

func (s *AuthService) SendVerification(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.EmailVerifiedAt != nil {
		return ErrEmailAlreadyVerified
	}

	_ = s.authTokenRepo.InvalidateUserTokensByType(ctx, user.ID, models.TokenTypeEmailVerification)

	rawToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return err
	}

	tokenHash := utils.HashToken(rawToken)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	tokenModel := &models.AuthToken{
		ID:        utils.NewULID(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		TokenType: models.TokenTypeEmailVerification,
		IsUsed:    false,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.authTokenRepo.Create(ctx, tokenModel); err != nil {
		return err
	}

	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.FrontendURL, rawToken)
	if s.mailer != nil {
		if err := s.mailer.SendVerificationEmail(ctx, user.Email, verifyLink); err != nil {
			slog.Error("Failed to send verification email", slog.String("email", user.Email), slog.Any("error", err))
		}
	}

	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, input models.VerifyEmailInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	tokenHash := utils.HashToken(input.Token)
	storedToken, err := s.authTokenRepo.GetByHashAndType(ctx, tokenHash, models.TokenTypeEmailVerification)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAuthTokenInvalid
		}
		return err
	}

	if storedToken.IsUsed {
		return ErrAuthTokenUsed
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return ErrAuthTokenInvalid
	}

	// Update user email_verified_at and activate status
	if err := s.userRepo.VerifyEmail(ctx, storedToken.UserID); err != nil {
		return err
	}

	// Mark token as used
	return s.authTokenRepo.MarkAsUsed(ctx, storedToken.ID)
}

func (s *AuthService) ResendVerification(ctx context.Context, input models.ResendVerificationInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		// Anti user-enumeration: return nil if email not found
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	if user.EmailVerifiedAt != nil {
		return ErrEmailAlreadyVerified
	}

	return s.SendVerification(ctx, user.ID)
}
