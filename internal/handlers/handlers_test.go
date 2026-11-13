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
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
	"github.com/mochammadbagas/go-crud-api/internal/service"
)

// In-memory repositories for handler testing
type testUserRepo struct {
	user *models.User
}

func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (r *testUserRepo) Count(ctx context.Context) (int64, error)             { return 1, nil }

type testSessionRepo struct {
	sessions map[string]*models.Session
}

func (r *testSessionRepo) Create(ctx context.Context, session *models.Session) error {
	r.sessions[session.ID] = session
	return nil
}
func (r *testSessionRepo) GetByID(ctx context.Context, id string) (*models.Session, error) {
	if s, ok := r.sessions[id]; ok {
		if s.IsExpired() {
			return nil, repository.ErrNotFound
		}
		return s, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testSessionRepo) Delete(ctx context.Context, id string) error {
	delete(r.sessions, id)
	return nil
}
func (r *testSessionRepo) DeleteExpired(ctx context.Context) error {
	for id, s := range r.sessions {
		if s.IsExpired() {
			delete(r.sessions, id)
		}
	}
	return nil
}

type testCategoryRepo struct {
	categories map[uint64]*models.Category
	bySlug     map[string]*models.Category
	nextID     uint64
}

func (r *testCategoryRepo) List(ctx context.Context) ([]models.Category, error) {
	var list []models.Category
	for _, c := range r.categories {
		list = append(list, *c)
	}
	return list, nil
}
func (r *testCategoryRepo) GetByID(ctx context.Context, id uint64) (*models.Category, error) {
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
	cat.ID = r.nextID
	r.nextID++
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Update(ctx context.Context, cat *models.Category) error {
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Delete(ctx context.Context, id uint64) error {
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
	posts  map[uint64]*models.Post
	bySlug map[string]*models.Post
	nextID uint64
}

func (r *testPostRepo) Create(ctx context.Context, post *models.Post) error {
	post.ID = r.nextID
	r.nextID++
	r.posts[post.ID] = post
	r.bySlug[post.Slug] = post
	return nil
}
func (r *testPostRepo) Update(ctx context.Context, post *models.Post) error {
	r.posts[post.ID] = post
	r.bySlug[post.Slug] = post
	return nil
}
func (r *testPostRepo) Delete(ctx context.Context, id uint64) error {
	if p, ok := r.posts[id]; ok {
		delete(r.bySlug, p.Slug)
		delete(r.posts, id)
		return nil
	}
	return repository.ErrNotFound
}
func (r *testPostRepo) GetByID(ctx context.Context, id uint64) (*models.Post, error) {
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
func (r *testPostRepo) IncrementViews(ctx context.Context, postID uint64) error {
	if p, ok := r.posts[postID]; ok {
		p.Views++
	}
	return nil
}
func (r *testPostRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.posts)), nil
}

func setupTestServer(t *testing.T) (http.Handler, string) {
	hashedPassword, err := service.HashPassword("secret123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	adminUser := &models.User{
		ID:           1,
		Name:         "Admin Test",
		Email:        "admin@test.com",
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	userRepo := &testUserRepo{user: adminUser}
	sessionRepo := &testSessionRepo{sessions: make(map[string]*models.Session)}
	catRepo := &testCategoryRepo{
		categories: make(map[uint64]*models.Category),
		bySlug:     make(map[string]*models.Category),
		nextID:     1,
	}
	postRepo := &testPostRepo{
		posts:  make(map[uint64]*models.Post),
		bySlug: make(map[string]*models.Post),
		nextID: 1,
	}

	cfg := &config.Config{
		Port:           "8080",
		CookieSecure:   false,
		UploadDir:      "./uploads",
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	markdownSvc := service.NewMarkdownService()
	authSvc := service.NewAuthService(userRepo, sessionRepo)
	categorySvc := service.NewCategoryService(catRepo)
	postSvc := service.NewPostService(postRepo, catRepo, markdownSvc)

	// Pre-seed a published article and a draft journal
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

	adminAuthHandler := handlers.NewAdminAuthHandler(authSvc, cfg)
	adminPostHandler := handlers.NewAdminPostHandler(postSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	adminUploadHandler := handlers.NewAdminUploadHandler(cfg)
	publicPostHandler := handlers.NewPublicPostHandler(postSvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, postSvc)

	router := routes.Setup(
		cfg,
		authSvc,
		adminAuthHandler,
		adminPostHandler,
		adminCategoryHandler,
		adminUploadHandler,
		publicPostHandler,
		publicCategoryHandler,
	)

	return router, "secret123"
}

func TestPublicEndpoints(t *testing.T) {
	router, _ := setupTestServer(t)

	t.Run("GET /api/v1/posts returns only published posts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got: %d", rec.Code)
		}

		var resp struct {
			Data []models.Post `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data) != 1 {
			t.Errorf("expected 1 published post, got %d", len(resp.Data))
		}
		if resp.Data[0].Slug != "published-go-article" {
			t.Errorf("expected slug 'published-go-article', got %q", resp.Data[0].Slug)
		}
	})

	t.Run("GET /api/v1/posts with invalid type returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?type=invalid_type", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid type, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/posts/{slug} renders HTML and computes SEO", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/published-go-article", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got: %d", rec.Code)
		}

		var post models.Post
		if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
			t.Fatalf("failed to decode post: %v", err)
		}

		if post.ContentHTML == "" {
			t.Errorf("expected ContentHTML to be rendered from markdown, got empty")
		}
		if post.SEOTitle != post.Title {
			t.Errorf("expected SEOTitle fallback to post.Title, got %q", post.SEOTitle)
		}
	})

	t.Run("GET /api/v1/posts/{slug} returns 404 for draft post (FR-021)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/draft-journal-entry", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for draft post on public endpoint, got %d", rec.Code)
		}
	})
}

func TestAdminAuthAndSession(t *testing.T) {
	router, password := setupTestServer(t)

	var sessionCookie *http.Cookie

	t.Run("POST /api/v1/admin/login sets HTTP-only cookie", func(t *testing.T) {
		loginPayload := map[string]string{
			"email":    "admin@test.com",
			"password": password,
		}
		body, _ := json.Marshal(loginPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got: %d: %s", rec.Code, rec.Body.String())
		}

		for _, cookie := range rec.Result().Cookies() {
			if cookie.Name == middleware.CookieSessionName {
				sessionCookie = cookie
				break
			}
		}

		if sessionCookie == nil {
			t.Fatal("expected session cookie in Set-Cookie header")
		}
		if !sessionCookie.HttpOnly {
			t.Errorf("expected cookie to have HttpOnly=true")
		}
	})

	t.Run("GET /api/v1/admin/me without session returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/admin/me with session cookie succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
		req.AddCookie(sessionCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var user models.UserProfile
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if user.Email != "admin@test.com" {
			t.Errorf("expected admin@test.com, got %q", user.Email)
		}
	})

	t.Run("POST /api/v1/admin/posts creates new post (draft by default)", func(t *testing.T) {
		createPayload := models.CreatePostInput{
			Type:    models.TypeBlog,
			Title:   "Admin Created Blog Post",
			Content: "Catatan blog baru dari admin.",
			Tags:    []string{"Personal", "Golang"},
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/posts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(sessionCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var createdPost models.Post
		_ = json.Unmarshal(rec.Body.Bytes(), &createdPost)
		if createdPost.Status != models.StatusDraft {
			t.Errorf("expected default status draft, got %q", createdPost.Status)
		}
		if createdPost.Slug != "admin-created-blog-post" {
			t.Errorf("expected slug 'admin-created-blog-post', got %q", createdPost.Slug)
		}
	})

	t.Run("POST /api/v1/admin/logout clears session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil)
		req.AddCookie(sessionCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		// Subsequent /me call should now fail with 401
		checkReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
		checkReq.AddCookie(sessionCookie)
		checkRec := httptest.NewRecorder()
		router.ServeHTTP(checkRec, checkReq)

		if checkRec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized after logout, got %d", checkRec.Code)
		}
	})

	t.Run("GET /api/v1/admin/me with expired session cookie returns 401", func(t *testing.T) {
		expiredCookie := &http.Cookie{
			Name:     middleware.CookieSessionName,
			Value:    "expired-session-id",
			Path:     "/",
			HttpOnly: true,
		}
		checkReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
		checkReq.AddCookie(expiredCookie)
		checkRec := httptest.NewRecorder()
		router.ServeHTTP(checkRec, checkReq)

		if checkRec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for expired session, got %d", checkRec.Code)
		}
	})
}

func TestAdminUploadMIMEValidation(t *testing.T) {
	router, password := setupTestServer(t)

	// Login to get valid session
	loginPayload := map[string]string{
		"email":    "admin@test.com",
		"password": password,
	}
	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == middleware.CookieSessionName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not found")
	}

	t.Run("Reject upload with fake image extension and plain text content", func(t *testing.T) {
		var reqBody bytes.Buffer
		writer := multipart.NewWriter(&reqBody)
		part, err := writer.CreateFormFile("image", "exploit.jpg")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		// Write plain text / script content instead of real image magic bytes
		_, _ = part.Write([]byte("<?php echo 'malicious payload'; ?>"))
		_ = writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/upload", &reqBody)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.AddCookie(sessionCookie)
		uploadRec := httptest.NewRecorder()
		router.ServeHTTP(uploadRec, uploadReq)

		if uploadRec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for spoofed image MIME, got %d", uploadRec.Code)
		}
	})

	t.Run("Accept upload with valid PNG magic bytes", func(t *testing.T) {
		var reqBody bytes.Buffer
		writer := multipart.NewWriter(&reqBody)
		part, err := writer.CreateFormFile("image", "valid.png")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		// Valid 1x1 transparent PNG magic bytes
		validPNG := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
			0x42, 0x60, 0x82,
		}
		_, _ = part.Write(validPNG)
		_ = writer.Close()

		uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/upload", &reqBody)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq.AddCookie(sessionCookie)
		uploadRec := httptest.NewRecorder()
		router.ServeHTTP(uploadRec, uploadReq)

		if uploadRec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created for valid PNG image, got %d: %s", uploadRec.Code, uploadRec.Body.String())
		}
	})
}

func TestAdminLoginRateLimit(t *testing.T) {
	router, _ := setupTestServer(t)

	loginPayload := []byte(`{"email":"admin@test.com","password":"wrong"}`)

	// First 10 requests should reach the handler and return 401
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.10:12345"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401 Unauthorized, got %d", i+1, rec.Code)
		}
	}

	// 11th request from same IP within window must be rate-limited (429)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests on 11th attempt, got %d", rec.Code)
	}
}


