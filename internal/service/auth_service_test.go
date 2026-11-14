package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

// mockUserRepo implements repository.UserRepository in-memory
type mockUserRepo struct {
	usersByID       map[string]*models.User
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		usersByID:       make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
	}
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if u, ok := m.usersByEmail[email]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if u, ok := m.usersByUsername[username]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	if u, ok := m.usersByID[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	m.usersByID[user.ID] = user
	m.usersByEmail[user.Email] = user
	m.usersByUsername[user.Username] = user
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	if u, ok := m.usersByID[id]; ok {
		u.PasswordHash = passwordHash
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockUserRepo) UpdateUsername(ctx context.Context, id string, username string) error {
	if u, ok := m.usersByID[id]; ok {
		delete(m.usersByUsername, u.Username)
		u.Username = username
		m.usersByUsername[username] = u
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if u, ok := m.usersByID[id]; ok {
		u.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockUserRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.usersByID)), nil
}

// mockProfileRepo implements repository.UserProfileRepository in-memory
type mockProfileRepo struct {
	profiles map[string]*models.UserProfile
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{
		profiles: make(map[string]*models.UserProfile),
	}
}

func (m *mockProfileRepo) GetByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	if p, ok := m.profiles[userID]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockProfileRepo) Create(ctx context.Context, profile *models.UserProfile) error {
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *mockProfileRepo) Update(ctx context.Context, profile *models.UserProfile) error {
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *mockProfileRepo) UpdateAvatar(ctx context.Context, userID string, avatarURL string) error {
	if p, ok := m.profiles[userID]; ok {
		p.AvatarURL = &avatarURL
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockProfileRepo) DeleteAvatar(ctx context.Context, userID string) error {
	if p, ok := m.profiles[userID]; ok {
		p.AvatarURL = nil
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockProfileRepo) GetUserWithProfile(ctx context.Context, userID string) (*models.UserWithProfile, error) {
	return nil, nil
}

// mockRefreshTokenRepo implements repository.RefreshTokenRepository in-memory
type mockRefreshTokenRepo struct {
	tokensByHash map[string]*models.RefreshToken
	tokensByID   map[string]*models.RefreshToken
}

func newMockRefreshTokenRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{
		tokensByHash: make(map[string]*models.RefreshToken),
		tokensByID:   make(map[string]*models.RefreshToken),
	}
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	m.tokensByHash[token.TokenHash] = token
	m.tokensByID[token.ID] = token
	return nil
}

func (m *mockRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	if t, ok := m.tokensByHash[tokenHash]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	if t, ok := m.tokensByID[id]; ok {
		t.IsRevoked = true
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockRefreshTokenRepo) RevokeFamily(ctx context.Context, tokenFamily string) error {
	for _, t := range m.tokensByID {
		if t.TokenFamily == tokenFamily {
			t.IsRevoked = true
		}
	}
	return nil
}

func (m *mockRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	for _, t := range m.tokensByID {
		if t.UserID == userID {
			t.IsRevoked = true
		}
	}
	return nil
}

func (m *mockRefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	return nil
}

func TestAuthService_FullLifecycleAndRTR(t *testing.T) {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	tokenRepo := newMockRefreshTokenRepo()
	cfg := &config.Config{
		JWTSecret:     "test-auth-secret",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	svc := NewAuthService(userRepo, profileRepo, tokenRepo, cfg)
	ctx := context.Background()

	// 1. Register User
	regInput := models.RegisterInput{
		Username: "john_doe",
		Email:    "john@example.com",
		Password: "password123",
		FullName: "John Doe",
	}

	user, err := svc.Register(ctx, regInput)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if user.ID == "" || len(user.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %q", user.ID)
	}
	if user.Role != models.RoleUser {
		t.Errorf("expected role 'user', got %q", user.Role)
	}

	// 2. Duplicate registration test
	_, err = svc.Register(ctx, regInput)
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists on duplicate registration, got %v", err)
	}

	// 3. Login
	loginInput := models.LoginInput{
		Email:    "john@example.com",
		Password: "password123",
	}

	tokenResp1, err := svc.Login(ctx, loginInput, "Mozilla/5.0", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}
	if tokenResp1.AccessToken == "" || tokenResp1.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens to be non-empty")
	}

	// 4. Refresh Token Rotation (RTR) - First rotation
	tokenResp2, err := svc.RefreshToken(ctx, tokenResp1.RefreshToken, "Mozilla/5.0", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to refresh token: %v", err)
	}
	if tokenResp2.RefreshToken == tokenResp1.RefreshToken {
		t.Error("expected new refresh token after rotation, got identical token")
	}

	// 5. REUSE DETECTION: Replaying tokenResp1.RefreshToken (which was already rotated/revoked)
	_, err = svc.RefreshToken(ctx, tokenResp1.RefreshToken, "Attacker/1.0", "192.168.1.100")
	if !errors.Is(err, ErrTokenReused) {
		t.Fatalf("expected ErrTokenReused when replaying revoked token, got %v", err)
	}

	// Verify that the whole family is now revoked!
	// Even tokenResp2 (the active one) should now be revoked!
	_, err = svc.RefreshToken(ctx, tokenResp2.RefreshToken, "Mozilla/5.0", "127.0.0.1")
	if !errors.Is(err, ErrTokenReused) {
		t.Errorf("expected tokenResp2 to be revoked after reuse detection, got %v", err)
	}
}
