package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type mockUserRepoForAuth struct {
	perms map[string][]string
}

func (m *mockUserRepoForAuth) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (m *mockUserRepoForAuth) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (m *mockUserRepoForAuth) GetByID(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}
func (m *mockUserRepoForAuth) Create(ctx context.Context, user *models.User) error { return nil }
func (m *mockUserRepoForAuth) CreateWithProfileAndRole(ctx context.Context, user *models.User, profile *models.UserProfile, roleID string) error {
	return nil
}
func (m *mockUserRepoForAuth) UpdatePassword(ctx context.Context, id string, hash string) error {
	return nil
}
func (m *mockUserRepoForAuth) UpdateUsername(ctx context.Context, id string, username string) error {
	return nil
}
func (m *mockUserRepoForAuth) UpdateStatus(ctx context.Context, id string, status string) error {
	return nil
}
func (m *mockUserRepoForAuth) VerifyEmail(ctx context.Context, id string) error               { return nil }
func (m *mockUserRepoForAuth) UpdateEmail(ctx context.Context, id string, email string) error { return nil }
func (m *mockUserRepoForAuth) Count(ctx context.Context) (int64, error)                       { return 0, nil }
func (m *mockUserRepoForAuth) Delete(ctx context.Context, id string) error                    { return nil }
func (m *mockUserRepoForAuth) UpdateByAdmin(ctx context.Context, id string, username, email string) error {
	return nil
}
func (m *mockUserRepoForAuth) ListAdmin(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepoForAuth) GetUserRoles(ctx context.Context, userID string) ([]*models.Role, error) {
	return nil, nil
}
func (m *mockUserRepoForAuth) GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (m *mockUserRepoForAuth) AssignRole(ctx context.Context, userID, roleID string) error { return nil }
func (m *mockUserRepoForAuth) RemoveRole(ctx context.Context, userID, roleID string) error { return nil }
func (m *mockUserRepoForAuth) SetRoles(ctx context.Context, userID string, roleIDs []string) error {
	return nil
}
func (m *mockUserRepoForAuth) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	return 1, nil
}
func (m *mockUserRepoForAuth) CountActiveSuperAdminsExcluding(ctx context.Context, userID string) (int64, error) {
	return 1, nil
}
func (m *mockUserRepoForAuth) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	return m.perms[userID], nil
}
func (m *mockUserRepoForAuth) CheckUserHasRole(ctx context.Context, userID, roleName string) (bool, error) {
	return false, nil
}
func (m *mockUserRepoForAuth) HasAssociatedData(ctx context.Context, userID string) (bool, error) {
	return false, nil
}

func TestJWTAuthMiddleware(t *testing.T) {
	secret := "test-secret-key-32-chars-long-123456"
	mw := middleware.JWTAuthMiddleware(secret)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserIDFromContext(r.Context())
		role := middleware.GetRoleFromContext(r.Context())
		if userID != "user-123" || role != "admin" {
			t.Errorf("unexpected context values: userID=%s, role=%s", userID, role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Missing header -> 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing token, got %d", rec.Code)
	}

	// Valid token -> 200
	token, err := utils.GenerateAccessToken(secret, time.Hour, "user-123", "testuser", "user@test.com", "admin")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", rec.Code)
	}
}

func TestPermissionCacheSweeperAndClear(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Sweeper runs and terminates cleanly
	done := make(chan struct{})
	go func() {
		middleware.StartPermissionCacheSweeper(ctx, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Stopped cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cache sweeper did not terminate cleanly upon context cancellation")
	}

	// Verify ClearUserPermissionCache does not panic
	middleware.ClearUserPermissionCache("user-123")
	middleware.ClearUserPermissionCache("")
}
