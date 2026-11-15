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
	ErrUserNotFound = errors.New("user not found")
)

type UserServiceInterface interface {
	GetProfile(ctx context.Context, userID string) (*models.UserWithProfile, error)
	UpdateProfile(ctx context.Context, userID string, input models.UpdateProfileInput) (*models.UserProfile, error)
	UpdateAvatar(ctx context.Context, userID string, avatarURL string) error
	DeleteAvatar(ctx context.Context, userID string) error
	UpdateUsername(ctx context.Context, userID string, input models.UpdateUsernameInput) error
	RequestEmailChange(ctx context.Context, userID string, input models.RequestEmailChangeInput) error
	ConfirmEmailChange(ctx context.Context, input models.ConfirmEmailChangeInput) error
}

type UserService struct {
	userRepo      repository.UserRepository
	profileRepo   repository.UserProfileRepository
	authTokenRepo repository.AuthTokenRepository
	mailer        mailer.Mailer
	cfg           *config.Config
}

func NewUserService(
	userRepo repository.UserRepository,
	profileRepo repository.UserProfileRepository,
	authTokenRepo repository.AuthTokenRepository,
	mail mailer.Mailer,
	cfg *config.Config,
) *UserService {
	return &UserService{
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		authTokenRepo: authTokenRepo,
		mailer:        mail,
		cfg:           cfg,
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.UserWithProfile, error) {
	profile, err := s.profileRepo.GetUserWithProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return profile, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, input models.UpdateProfileInput) (*models.UserProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existingProfile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			newProfile := &models.UserProfile{
				UserID:      userID,
				FullName:    input.FullName,
				Bio:         input.Bio,
				PhoneNumber: input.PhoneNumber,
			}
			if err := s.profileRepo.Create(ctx, newProfile); err != nil {
				return nil, err
			}
			return newProfile, nil
		}
		return nil, err
	}

	existingProfile.FullName = input.FullName
	existingProfile.Bio = input.Bio
	existingProfile.PhoneNumber = input.PhoneNumber

	if err := s.profileRepo.Update(ctx, existingProfile); err != nil {
		return nil, err
	}

	return existingProfile, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, userID string, avatarURL string) error {
	_, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = s.profileRepo.Create(ctx, &models.UserProfile{
				UserID:    userID,
				FullName:  "",
				AvatarURL: &avatarURL,
			})
			return nil
		}
		return err
	}

	return s.profileRepo.UpdateAvatar(ctx, userID, avatarURL)
}

func (s *UserService) DeleteAvatar(ctx context.Context, userID string) error {
	return s.profileRepo.DeleteAvatar(ctx, userID)
}

func (s *UserService) UpdateUsername(ctx context.Context, userID string, input models.UpdateUsernameInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	existingUser, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err == nil && existingUser != nil && existingUser.ID != userID {
		return ErrUsernameExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	return s.userRepo.UpdateUsername(ctx, userID, input.Username)
}

func (s *UserService) RequestEmailChange(ctx context.Context, userID string, input models.RequestEmailChangeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.Email == input.NewEmail {
		return errors.New("new email cannot be the same as current email")
	}

	// Verify current password for sensitive email change operation
	match, err := argon2id.ComparePasswordAndHash(input.Password, user.PasswordHash)
	if err != nil || !match {
		return errors.New("incorrect confirmation password")
	}

	// Ensure new email is not already taken by another user
	existing, err := s.userRepo.GetByEmail(ctx, input.NewEmail)
	if err == nil && existing != nil {
		return ErrEmailExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	// Invalidate any previous email_change tokens for this user
	_ = s.authTokenRepo.InvalidateUserTokensByType(ctx, userID, models.TokenTypeEmailChange)

	rawToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return err
	}

	tokenHash := utils.HashToken(rawToken)
	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	newEmailPayload := input.NewEmail

	tokenModel := &models.AuthToken{
		ID:        utils.NewULID(),
		UserID:    userID,
		TokenHash: tokenHash,
		TokenType: models.TokenTypeEmailChange,
		Payload:   &newEmailPayload,
		IsUsed:    false,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.authTokenRepo.Create(ctx, tokenModel); err != nil {
		return err
	}

	confirmLink := fmt.Sprintf("%s/confirm-email-change?token=%s", s.cfg.FrontendURL, rawToken)
	if s.mailer != nil {
		if err := s.mailer.SendEmailChangeEmail(ctx, input.NewEmail, confirmLink); err != nil {
			slog.Error("Failed to send email change confirmation email", slog.String("email", input.NewEmail), slog.Any("error", err))
		}
	}

	return nil
}

func (s *UserService) ConfirmEmailChange(ctx context.Context, input models.ConfirmEmailChangeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	tokenHash := utils.HashToken(input.Token)
	storedToken, err := s.authTokenRepo.GetByHashAndType(ctx, tokenHash, models.TokenTypeEmailChange)
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

	if storedToken.Payload == nil || *storedToken.Payload == "" {
		return errors.New("invalid new email payload in token")
	}

	newEmail := *storedToken.Payload

	// Final duplicate check in case another user registered this email while token was pending
	existing, err := s.userRepo.GetByEmail(ctx, newEmail)
	if err == nil && existing != nil && existing.ID != storedToken.UserID {
		return ErrEmailExists
	}

	// Update email and mark email as verified
	if err := s.userRepo.UpdateEmail(ctx, storedToken.UserID, newEmail); err != nil {
		return err
	}

	// Mark token as used
	return s.authTokenRepo.MarkAsUsed(ctx, storedToken.ID)
}
