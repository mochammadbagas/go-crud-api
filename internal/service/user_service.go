package service

import (
	"context"
	"errors"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

var (
	ErrUserNotFound = errors.New("pengguna tidak ditemukan")
)

type UserServiceInterface interface {
	GetProfile(ctx context.Context, userID string) (*models.UserWithProfile, error)
	UpdateProfile(ctx context.Context, userID string, input models.UpdateProfileInput) (*models.UserProfile, error)
	UpdateAvatar(ctx context.Context, userID string, avatarURL string) error
	DeleteAvatar(ctx context.Context, userID string) error
	UpdateUsername(ctx context.Context, userID string, input models.UpdateUsernameInput) error
}

type UserService struct {
	userRepo    repository.UserRepository
	profileRepo repository.UserProfileRepository
}

func NewUserService(userRepo repository.UserRepository, profileRepo repository.UserProfileRepository) *UserService {
	return &UserService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
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

	// Ensure profile exists; if not, create it
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
	// Ensure profile exists
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

	// Check if username is already taken by someone else
	existingUser, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err == nil && existingUser != nil && existingUser.ID != userID {
		return ErrUsernameExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	return s.userRepo.UpdateUsername(ctx, userID, input.Username)
}
