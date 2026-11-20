package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

func TestUserService_EmailChangeWorkflow(t *testing.T) {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	authTokenRepo := newMockAuthTokenRepo()
	refreshTokenRepo := newMockRefreshTokenRepo()
	mailSvc := mailer.NewLogMailer()
	cfg := &config.Config{
		FrontendURL: "http://localhost:3000",
	}

	svc := NewUserService(userRepo, profileRepo, authTokenRepo, refreshTokenRepo, mailSvc, cfg)
	ctx := context.Background()

	// Seed user with hashed password
	hash, _ := argon2id.CreateHash("password123", argon2id.DefaultParams)
	user := &models.User{
		ID:           utils.NewULID(),
		Username:     "testuser",
		Email:        "current@example.com",
		PasswordHash: hash,
		Role:         models.RoleUser,
		Status:       models.StatusActive,
	}
	_ = userRepo.Create(ctx, user)

	// 1. Request change email with wrong password
	err := svc.RequestEmailChange(ctx, user.ID, models.RequestEmailChangeInput{
		NewEmail: "new@example.com",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Fatal("expected error on incorrect password")
	}

	// 2. Request change email with valid credentials
	err = svc.RequestEmailChange(ctx, user.ID, models.RequestEmailChangeInput{
		NewEmail: "new@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to request email change: %v", err)
	}

	// Locate token in authTokenRepo
	var foundToken *models.AuthToken
	for _, tok := range authTokenRepo.tokens {
		if tok.UserID == user.ID && tok.TokenType == models.TokenTypeEmailChange {
			foundToken = tok
			break
		}
	}
	if foundToken == nil {
		t.Fatal("expected email change token in repo")
	}
	if *foundToken.Payload != "new@example.com" {
		t.Errorf("expected payload 'new@example.com', got %q", *foundToken.Payload)
	}

	// 3. Confirm with bogus token
	err = svc.ConfirmEmailChange(ctx, models.ConfirmEmailChangeInput{
		Token: "invalid-token",
	})
	if !errors.Is(err, ErrAuthTokenInvalid) {
		t.Errorf("expected ErrAuthTokenInvalid for bogus token, got %v", err)
	}

	// 4. Confirm with valid token by creating a mock token with known raw token
	rawToken, _ := utils.GenerateRefreshToken()
	tokenHash := utils.HashToken(rawToken)
	newEmail := "confirmed@example.com"
	_ = authTokenRepo.Create(ctx, &models.AuthToken{
		ID:        utils.NewULID(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		TokenType: models.TokenTypeEmailChange,
		Payload:   &newEmail,
		IsUsed:    false,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		CreatedAt: time.Now().UTC(),
	})

	err = svc.ConfirmEmailChange(ctx, models.ConfirmEmailChangeInput{
		Token: rawToken,
	})
	if err != nil {
		t.Fatalf("failed to confirm email change: %v", err)
	}

	// Check updated user in userRepo
	updatedUser, _ := userRepo.GetByID(ctx, user.ID)
	if updatedUser.Email != "confirmed@example.com" {
		t.Errorf("expected user email to be 'confirmed@example.com', got %q", updatedUser.Email)
	}
	if updatedUser.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set after email change confirmation")
	}
}

type errMockProfileRepo struct {
	*mockProfileRepo
	createErr error
}

func (m *errMockProfileRepo) Create(ctx context.Context, p *models.UserProfile) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.mockProfileRepo.Create(ctx, p)
}

func TestUserService_UpdateAvatar_ErrorPropagation(t *testing.T) {
	userRepo := newMockUserRepo()
	expectedErr := errors.New("db insert failure")
	profileRepo := &errMockProfileRepo{
		mockProfileRepo: newMockProfileRepo(),
		createErr:       expectedErr,
	}
	svc := NewUserService(userRepo, profileRepo, nil, nil, nil, nil)

	err := svc.UpdateAvatar(context.Background(), "user-123", "http://example.com/avatar.png")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

type errMockUserRepo struct {
	*mockUserRepo
	updateUsernameErr error
	updateEmailErr    error
}

func (m *errMockUserRepo) UpdateUsername(ctx context.Context, id, username string) error {
	if m.updateUsernameErr != nil {
		return m.updateUsernameErr
	}
	return m.mockUserRepo.UpdateUsername(ctx, id, username)
}

func (m *errMockUserRepo) UpdateEmail(ctx context.Context, id, newEmail string) error {
	if m.updateEmailErr != nil {
		return m.updateEmailErr
	}
	return m.mockUserRepo.UpdateEmail(ctx, id, newEmail)
}

func TestUserService_UpdateUsername_DuplicateHandling(t *testing.T) {
	baseUserRepo := newMockUserRepo()
	userRepo := &errMockUserRepo{
		mockUserRepo:      baseUserRepo,
		updateUsernameErr: repository.ErrDuplicate,
	}
	svc := NewUserService(userRepo, newMockProfileRepo(), nil, nil, nil, nil)

	err := svc.UpdateUsername(context.Background(), "user-1", models.UpdateUsernameInput{
		Username: "newname",
	})
	if !errors.Is(err, ErrUsernameExists) {
		t.Fatalf("expected ErrUsernameExists, got %v", err)
	}
}
