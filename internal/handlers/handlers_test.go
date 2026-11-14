package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/handlers"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

type testUserRepo struct {
	usersByID       map[string]*models.User
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{
		usersByID:       make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
	}
}

func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if u, ok := r.usersByEmail[email]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if u, ok := r.usersByUsername[username]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	if u, ok := r.usersByID[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) Create(ctx context.Context, user *models.User) error {
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByUsername[user.Username] = user
	return nil
}
func (r *testUserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	if u, ok := r.usersByID[id]; ok {
		u.PasswordHash = passwordHash
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateUsername(ctx context.Context, id string, username string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByUsername, u.Username)
		u.Username = username
		r.usersByUsername[username] = u
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if u, ok := r.usersByID[id]; ok {
		u.Status = status
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) VerifyEmail(ctx context.Context, id string) error {
	if u, ok := r.usersByID[id]; ok {
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
		u.Status = models.StatusActive
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateEmail(ctx context.Context, id string, newEmail string) error {
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
func (r *testUserRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.usersByID)), nil
}

type testAuthTokenRepo struct {
	tokens map[string]*models.AuthToken
}

func newTestAuthTokenRepo() *testAuthTokenRepo {
	return &testAuthTokenRepo{tokens: make(map[string]*models.AuthToken)}
}

func (r *testAuthTokenRepo) Create(ctx context.Context, token *models.AuthToken) error {
	r.tokens[token.TokenHash+":"+token.TokenType] = token
	return nil
}

func (r *testAuthTokenRepo) GetByHashAndType(ctx context.Context, tokenHash, tokenType string) (*models.AuthToken, error) {
	if t, ok := r.tokens[tokenHash+":"+tokenType]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testAuthTokenRepo) MarkAsUsed(ctx context.Context, id string) error {
	for _, t := range r.tokens {
		if t.ID == id {
			t.IsUsed = true
			return nil
		}
	}
	return nil
}

func (r *testAuthTokenRepo) InvalidateUserTokensByType(ctx context.Context, userID, tokenType string) error {
	for _, t := range r.tokens {
		if t.UserID == userID && t.TokenType == tokenType {
			t.IsUsed = true
		}
	}
	return nil
}

func (r *testAuthTokenRepo) DeleteExpiredOrUsed(ctx context.Context) error {
	return nil
}

type testProfileRepo struct {
	profiles map[string]*models.UserProfile
}

func newTestProfileRepo() *testProfileRepo {
	return &testProfileRepo{profiles: make(map[string]*models.UserProfile)}
}

func (r *testProfileRepo) GetByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	if p, ok := r.profiles[userID]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testProfileRepo) Create(ctx context.Context, profile *models.UserProfile) error {
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *testProfileRepo) Update(ctx context.Context, profile *models.UserProfile) error {
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *testProfileRepo) UpdateAvatar(ctx context.Context, userID string, avatarURL string) error {
	if p, ok := r.profiles[userID]; ok {
		p.AvatarURL = &avatarURL
		return nil
	}
	return repository.ErrNotFound
}
func (r *testProfileRepo) DeleteAvatar(ctx context.Context, userID string) error {
	if p, ok := r.profiles[userID]; ok {
		p.AvatarURL = nil
		return nil
	}
	return repository.ErrNotFound
}
func (r *testProfileRepo) GetUserWithProfile(ctx context.Context, userID string) (*models.UserWithProfile, error) {
	p := r.profiles[userID]
	fullName := ""
	var bio, phone, avatar *string
	if p != nil {
		fullName = p.FullName
		bio = p.Bio
		phone = p.PhoneNumber
		avatar = p.AvatarURL
	}
	return &models.UserWithProfile{
		ID:          userID,
		Username:    "admin",
		Email:       "admin@test.com",
		Role:        models.RoleAdmin,
		Status:      models.StatusActive,
		FullName:    fullName,
		Bio:         bio,
		PhoneNumber: phone,
		AvatarURL:   avatar,
	}, nil
}

type testRefreshTokenRepo struct {
	tokensByHash map[string]*models.RefreshToken
	tokensByID   map[string]*models.RefreshToken
}

func newTestRefreshTokenRepo() *testRefreshTokenRepo {
	return &testRefreshTokenRepo{
		tokensByHash: make(map[string]*models.RefreshToken),
		tokensByID:   make(map[string]*models.RefreshToken),
	}
}

func (r *testRefreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	r.tokensByHash[token.TokenHash] = token
	r.tokensByID[token.ID] = token
	return nil
}
func (r *testRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	if t, ok := r.tokensByHash[tokenHash]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testRefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	if t, ok := r.tokensByID[id]; ok {
		t.IsRevoked = true
		return nil
	}
	return repository.ErrNotFound
}
func (r *testRefreshTokenRepo) RevokeFamily(ctx context.Context, tokenFamily string) error {
	for _, t := range r.tokensByID {
		if t.TokenFamily == tokenFamily {
			t.IsRevoked = true
		}
	}
	return nil
}
func (r *testRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	for _, t := range r.tokensByID {
		if t.UserID == userID {
			t.IsRevoked = true
		}
	}
	return nil
}
func (r *testRefreshTokenRepo) DeleteExpired(ctx context.Context) error { return nil }

type testCategoryRepo struct {
	categories map[string]*models.Category
	bySlug     map[string]*models.Category
}

func newTestCategoryRepo() *testCategoryRepo {
	return &testCategoryRepo{
		categories: make(map[string]*models.Category),
		bySlug:     make(map[string]*models.Category),
	}
}

func (r *testCategoryRepo) List(ctx context.Context) ([]models.Category, error) {
	var list []models.Category
	for _, c := range r.categories {
		list = append(list, *c)
	}
	return list, nil
}
func (r *testCategoryRepo) GetByID(ctx context.Context, id string) (*models.Category, error) {
	if c, ok := r.categories[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testCategoryRepo) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	if c, ok := r.bySlug[slug]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testCategoryRepo) Create(ctx context.Context, cat *models.Category) error {
	if cat.ID == "" {
		cat.ID = utils.NewULID()
	}
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Update(ctx context.Context, cat *models.Category) error {
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Delete(ctx context.Context, id string) error {
	if c, ok := r.categories[id]; ok {
		delete(r.bySlug, c.Slug)
		delete(r.categories, id)
		return nil
	}
	return repository.ErrNotFound
}
func (r *testCategoryRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.categories)), nil
}

type testPostRepo struct {
	posts  map[string]*models.Post
	bySlug map[string]*models.Post
}

func newTestPostRepo() *testPostRepo {
	return &testPostRepo{
		posts:  make(map[string]*models.Post),
		bySlug: make(map[string]*models.Post),
	}
}

func (r *testPostRepo) Create(ctx context.Context, post *models.Post) error {
	if post.ID == "" {
		post.ID = utils.NewULID()
	}
	r.posts[post.ID] = post
	r.bySlug[post.Slug] = post
	return nil
}
func (r *testPostRepo) Update(ctx context.Context, post *models.Post) error {
	r.posts[post.ID] = post
	r.bySlug[post.Slug] = post
	return nil
}
func (r *testPostRepo) Delete(ctx context.Context, id string) error {
	if p, ok := r.posts[id]; ok {
		delete(r.bySlug, p.Slug)
		delete(r.posts, id)
		return nil
	}
	return repository.ErrNotFound
}
func (r *testPostRepo) GetByID(ctx context.Context, id string) (*models.Post, error) {
	if p, ok := r.posts[id]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testPostRepo) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	if p, ok := r.bySlug[slug]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testPostRepo) GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error) {
	if p, ok := r.bySlug[slug]; ok && p.Status == models.StatusPublished {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testPostRepo) ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	var list []models.Post
	for _, p := range r.posts {
		if p.Status == models.StatusPublished {
			list = append(list, *p)
		}
	}
	return list, int64(len(list)), nil
}
func (r *testPostRepo) ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	var list []models.Post
	for _, p := range r.posts {
		list = append(list, *p)
	}
	return list, int64(len(list)), nil
}
func (r *testPostRepo) IncrementViews(ctx context.Context, postID string) error {
	if p, ok := r.posts[postID]; ok {
		p.Views++
	}
	return nil
}
func (r *testPostRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.posts)), nil
}

func setupTestServer(t *testing.T) (http.Handler, string, string) {
	hashedPassword, err := argon2id.CreateHash("secret123", argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	adminID := utils.NewULID()
	adminUser := &models.User{
		ID:           adminID,
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: hashedPassword,
		Role:         models.RoleAdmin,
		Status:       models.StatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	userRepo := newTestUserRepo()
	_ = userRepo.Create(context.Background(), adminUser)

	profileRepo := newTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &models.UserProfile{
		UserID:   adminID,
		FullName: "Administrator",
	})

	refreshTokenRepo := newTestRefreshTokenRepo()
	authTokenRepo := newTestAuthTokenRepo()
	catRepo := newTestCategoryRepo()
	postRepo := newTestPostRepo()

	cfg := &config.Config{
		Port:           "8080",
		CookieSecure:   false,
		UploadDir:      "./uploads",
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTSecret:      "test-starter-kit-secret-key-12345",
		JWTAccessTTL:   15 * time.Minute,
		JWTRefreshTTL:  7 * 24 * time.Hour,
		FrontendURL:    "http://localhost:3000",
		MailerDriver:   "log",
	}

	mailSvc := mailer.NewLogMailer()
	markdownSvc := service.NewMarkdownService()
	authSvc := service.NewAuthService(userRepo, profileRepo, refreshTokenRepo, authTokenRepo, mailSvc, cfg)
	userSvc := service.NewUserService(userRepo, profileRepo, authTokenRepo, mailSvc, cfg)
	categorySvc := service.NewCategoryService(catRepo)
	postSvc := service.NewPostService(postRepo, catRepo, markdownSvc)

	now := time.Now().UTC()
	_ = postRepo.Create(context.Background(), &models.Post{
		Type:        models.TypeArticle,
		Title:       "Published Go Article",
		Slug:        "published-go-article",
		Content:     "# Welcome\n\nThis is a published post.",
		Status:      models.StatusPublished,
		PublishedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	_ = postRepo.Create(context.Background(), &models.Post{
		Type:      models.TypeJournal,
		Title:     "Draft Journal Entry",
		Slug:      "draft-journal-entry",
		Content:   "Secret WIP journal entry.",
		Status:    models.StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	})

	authHandler := handlers.NewAuthHandler(authSvc, cfg)
	userHandler := handlers.NewUserHandler(userSvc, cfg)
	adminPostHandler := handlers.NewAdminPostHandler(postSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	adminUploadHandler := handlers.NewAdminUploadHandler(cfg)
	publicPostHandler := handlers.NewPublicPostHandler(postSvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, postSvc)

	router := routes.Setup(
		cfg,
		authHandler,
		userHandler,
		adminPostHandler,
		adminCategoryHandler,
		adminUploadHandler,
		publicPostHandler,
		publicCategoryHandler,
	)

	// Pre-generate an admin JWT token
	adminToken, err := utils.GenerateAccessToken(cfg.JWTSecret, cfg.JWTAccessTTL, adminID, "admin", "admin@test.com", models.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	return router, adminToken, "secret123"
}

func TestPublicEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", rec.Code)
	}

	// 2. Public posts list
	req = httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/posts, got %d", rec.Code)
	}

	// 3. Public get published post by slug
	req = httptest.NewRequest(http.MethodGet, "/api/v1/posts/published-go-article", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for published post, got %d", rec.Code)
	}

	// 4. Draft post must return 404 on public route
	req = httptest.NewRequest(http.MethodGet, "/api/v1/posts/draft-journal-entry", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for draft post accessed publicly, got %d", rec.Code)
	}
}

func TestAuthEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Register new user
	regPayload := `{"username":"alice_user","email":"alice@test.com","password":"mypassword123","full_name":"Alice Wonderland"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for register, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2. Login user
	loginPayload := `{"email":"alice@test.com","password":"mypassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for login, got %d (%s)", rec.Code, rec.Body.String())
	}

	var tokenResp models.TokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens to be present")
	}

	// Verify Cookie was set
	cookies := rec.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == handlers.RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected refresh_token cookie to be set")
	}

	// 3. User profile /api/v1/users/me using access token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /users/me, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 4. Refresh token rotation
	refreshPayload := `{"refresh_token":"` + tokenResp.RefreshToken + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for refresh token, got %d (%s)", rec.Code, rec.Body.String())
	}

	var rotatedResp models.TokenResponse
	_ = json.NewDecoder(rec.Body).Decode(&rotatedResp)
	if rotatedResp.AccessToken == "" || rotatedResp.RefreshToken == tokenResp.RefreshToken {
		t.Errorf("expected rotated tokens, got same or empty")
	}
}

func TestAdminEndpointsWithJWT(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. Create Category as Admin
	catPayload := `{"name":"Backend Architecture"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/categories", bytes.NewBufferString(catPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin create category, got %d (%s)", rec.Code, rec.Body.String())
	}

	var createdCat models.Category
	_ = json.NewDecoder(rec.Body).Decode(&createdCat)
	if createdCat.ID == "" || len(createdCat.ID) != 26 {
		t.Errorf("expected 26-char ULID for category ID, got %q", createdCat.ID)
	}

	// 2. Create Post as Admin with ULID Category
	postPayload := `{
		"type": "article",
		"title": "Mastering ULID in Go",
		"content": "Why ULID is better than auto increment.",
		"category_id": "` + createdCat.ID + `",
		"status": "published"
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/posts", bytes.NewBufferString(postPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin create post, got %d (%s)", rec.Code, rec.Body.String())
	}

	var createdPost models.Post
	_ = json.NewDecoder(rec.Body).Decode(&createdPost)
	if createdPost.ID == "" || len(createdPost.ID) != 26 {
		t.Errorf("expected 26-char ULID for post ID, got %q", createdPost.ID)
	}

	// 3. Unauthorized access without token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/posts", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth header, got %d", rec.Code)
	}
}

func TestUploadEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// Create multipart image payload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	// Minimal valid PNG header magic bytes
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	_, _ = part.Write(pngHeader)
	_, _ = part.Write(make([]byte, 100))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for admin upload, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestForgotPasswordAndResetPasswordEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Forgot password request (anti user-enumeration should return 200 for any valid email)
	payload := `{"email":"admin@mmdbconnect.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for forgot password, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2. Reset password with invalid token
	resetPayload := `{"token":"invalid-token","new_password":"newpassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(resetPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for reset password with invalid token, got %d", rec.Code)
	}
}

func TestEmailVerificationAndChangeEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. Verify email with empty/invalid token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=invalid-token", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for verify email with invalid token, got %d", rec.Code)
	}

	// 2. Resend verification
	resendPayload := `{"email":"admin@mmdbconnect.com"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewBufferString(resendPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Errorf("unexpected status code for resend verification: %d", rec.Code)
	}

	// 3. Request email change without auth -> 401
	changeReq := `{"new_email":"newadmin@example.com","password":"secret123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/email/request-change", bytes.NewBufferString(changeReq))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth for request email change, got %d", rec.Code)
	}

	// 4. Request email change with valid auth
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/email/request-change", bytes.NewBufferString(changeReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for request email change with auth, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 5. Confirm email change with bogus token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/email/confirm-change?token=bogus", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for confirm email change with bogus token, got %d", rec.Code)
	}
}
