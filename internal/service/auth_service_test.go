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
)

// mockUserRepo implements repository.UserRepository in-memory
type mockUserRepo struct {
	usersByID       map[string]*models.User
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
	userRoles       map[string][]string
	userRoleIDs     map[string][]string
	userPerms       map[string][]string
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		usersByID:       make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		userRoles:       make(map[string][]string),
		userRoleIDs:     make(map[string][]string),
		userPerms:       make(map[string][]string),
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

func (m *mockUserRepo) CreateWithProfileAndRole(ctx context.Context, user *models.User, profile *models.UserProfile, roleID string) error {
	m.usersByID[user.ID] = user
	m.usersByEmail[user.Email] = user
	m.usersByUsername[user.Username] = user
	if roleID != "" {
		m.userRoleIDs[user.ID] = append(m.userRoleIDs[user.ID], roleID)
	}
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

func (r *mockUserRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if u, ok := r.usersByID[id]; ok {
		u.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockUserRepo) VerifyEmail(ctx context.Context, id string) error {
	if u, ok := r.usersByID[id]; ok {
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
		u.Status = models.StatusActive
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockUserRepo) UpdateEmail(ctx context.Context, id string, newEmail string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByEmail, u.Email)
		u.Email = newEmail
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
		r.usersByEmail[newEmail] = u
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockUserRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.usersByID)), nil
}

func (r *mockUserRepo) Delete(ctx context.Context, id string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByID, id)
		delete(r.usersByEmail, u.Email)
		delete(r.usersByUsername, u.Username)
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockUserRepo) UpdateByAdmin(ctx context.Context, id string, username, email string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByUsername, u.Username)
		delete(r.usersByEmail, u.Email)
		u.Username = username
		u.Email = email
		r.usersByUsername[username] = u
		r.usersByEmail[email] = u
		return nil
	}
	return repository.ErrNotFound
}

func (r *mockUserRepo) ListAdmin(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error) {
	var list []*models.UserWithProfile
	for _, u := range r.usersByID {
		roles := r.userRoles[u.ID]
		if len(roles) == 0 {
			roles = []string{u.Role}
		}
		list = append(list, &models.UserWithProfile{
			ID:              u.ID,
			Username:        u.Username,
			Email:           u.Email,
			Role:            u.Role,
			Roles:           roles,
			Status:          u.Status,
			EmailVerifiedAt: u.EmailVerifiedAt,
			CreatedAt:       u.CreatedAt,
			UpdatedAt:       u.UpdatedAt,
		})
	}
	return list, int64(len(list)), nil
}

func (r *mockUserRepo) GetUserRoles(ctx context.Context, userID string) ([]*models.Role, error) {
	roleNames := r.userRoles[userID]
	var res []*models.Role
	for _, rn := range roleNames {
		res = append(res, &models.Role{ID: rn, Name: rn, DisplayName: rn})
	}
	return res, nil
}

func (r *mockUserRepo) GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	if roles, ok := r.userRoles[userID]; ok && len(roles) > 0 {
		return roles, nil
	}
	if u, ok := r.usersByID[userID]; ok {
		return []string{u.Role}, nil
	}
	return []string{}, nil
}

func (r *mockUserRepo) AssignRole(ctx context.Context, userID, roleID string) error {
	r.userRoleIDs[userID] = append(r.userRoleIDs[userID], roleID)
	r.userRoles[userID] = append(r.userRoles[userID], roleID)
	return nil
}

func (r *mockUserRepo) RemoveRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (r *mockUserRepo) SetRoles(ctx context.Context, userID string, roleIDs []string) error {
	r.userRoleIDs[userID] = roleIDs
	r.userRoles[userID] = roleIDs
	return nil
}

func (r *mockUserRepo) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	for _, u := range r.usersByID {
		if u.Status == models.StatusActive {
			roles := r.userRoles[u.ID]
			isSuper := (u.Role == models.RoleSuperAdmin)
			for _, r := range roles {
				if r == models.RoleSuperAdmin {
					isSuper = true
				}
			}
			if isSuper {
				count++
			}
		}
	}
	return count, nil
}

func (r *mockUserRepo) CountActiveSuperAdminsExcluding(ctx context.Context, userID string) (int64, error) {
	var count int64
	for _, u := range r.usersByID {
		if u.ID != userID && u.Status == models.StatusActive {
			roles := r.userRoles[u.ID]
			isSuper := (u.Role == models.RoleSuperAdmin)
			for _, r := range roles {
				if r == models.RoleSuperAdmin {
					isSuper = true
				}
			}
			if isSuper {
				count++
			}
		}
	}
	return count, nil
}

func (r *mockUserRepo) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	return r.userPerms[userID], nil
}

func (r *mockUserRepo) CheckUserHasRole(ctx context.Context, userID, roleName string) (bool, error) {
	roles := r.userRoles[userID]
	for _, r := range roles {
		if r == roleName {
			return true, nil
		}
	}
	if u, ok := r.usersByID[userID]; ok && u.Role == roleName {
		return true, nil
	}
	return false, nil
}

func (r *mockUserRepo) HasAssociatedData(ctx context.Context, userID string) (bool, error) {
	return false, nil
}

// mockAuthTokenRepo implements repository.AuthTokenRepository in-memory
type mockAuthTokenRepo struct {
	tokens map[string]*models.AuthToken
}

func newMockAuthTokenRepo() *mockAuthTokenRepo {
	return &mockAuthTokenRepo{
		tokens: make(map[string]*models.AuthToken),
	}
}

func (m *mockAuthTokenRepo) Create(ctx context.Context, token *models.AuthToken) error {
	m.tokens[token.TokenHash+":"+token.TokenType] = token
	return nil
}

func (m *mockAuthTokenRepo) GetByHashAndType(ctx context.Context, tokenHash, tokenType string) (*models.AuthToken, error) {
	if t, ok := m.tokens[tokenHash+":"+tokenType]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockAuthTokenRepo) MarkAsUsed(ctx context.Context, id string) error {
	for _, t := range m.tokens {
		if t.ID == id {
			t.IsUsed = true
			return nil
		}
	}
	return nil
}

func (m *mockAuthTokenRepo) MarkAsUsedAtomic(ctx context.Context, id string) (bool, error) {
	for _, t := range m.tokens {
		if t.ID == id {
			if t.IsUsed {
				return false, nil
			}
			t.IsUsed = true
			return true, nil
		}
	}
	return false, nil
}

func (m *mockAuthTokenRepo) InvalidateUserTokensByType(ctx context.Context, userID, tokenType string) error {
	for _, t := range m.tokens {
		if t.UserID == userID && t.TokenType == tokenType {
			t.IsUsed = true
		}
	}
	return nil
}

func (m *mockAuthTokenRepo) DeleteExpiredOrUsed(ctx context.Context) error {
	return nil
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

func (m *mockRefreshTokenRepo) RevokeAtomic(ctx context.Context, id string) (bool, error) {
	if t, ok := m.tokensByID[id]; ok {
		if t.IsRevoked {
			return false, nil
		}
		t.IsRevoked = true
		return true, nil
	}
	return false, repository.ErrNotFound
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
	authTokenRepo := newMockAuthTokenRepo()
	mailSvc := mailer.NewLogMailer()
	cfg := &config.Config{
		JWTSecret:     "test-auth-secret",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		FrontendURL:   "http://localhost:3000",
	}

	svc := NewAuthService(userRepo, profileRepo, tokenRepo, authTokenRepo, mailSvc, cfg)
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

	// 6. Forgot Password (Anti user-enumeration check)
	err = svc.ForgotPassword(ctx, models.ForgotPasswordInput{Email: "nonexistent@example.com"})
	if err != nil {
		t.Errorf("expected nil on non-existent email for anti-enumeration, got %v", err)
	}

	err = svc.ForgotPassword(ctx, models.ForgotPasswordInput{Email: "john@example.com"})
	if err != nil {
		t.Fatalf("failed to request forgot password: %v", err)
	}

	// Locate the generated reset token from mockAuthTokenRepo
	var resetRawTokenHash string
	for _, tok := range authTokenRepo.tokens {
		if tok.UserID == user.ID && tok.TokenType == models.TokenTypePasswordReset {
			resetRawTokenHash = tok.TokenHash
			break
		}
	}
	if resetRawTokenHash == "" {
		t.Fatal("expected reset token to be generated in authTokenRepo")
	}

	// 7. Reset Password
	// Find raw token representation or verify with wrong token
	err = svc.ResetPassword(ctx, models.ResetPasswordInput{
		Token:       "invalid-token",
		NewPassword: "newpassword123",
	})
	if !errors.Is(err, ErrAuthTokenInvalid) {
		t.Errorf("expected ErrAuthTokenInvalid for bogus token, got %v", err)
	}
}
