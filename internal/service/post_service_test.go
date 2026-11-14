package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// mockPostRepo implements repository.PostRepository in memory for unit testing
type mockPostRepo struct {
	posts             map[string]*models.Post
	bySlug            map[string]*models.Post
	getPublishedCalls int
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{
		posts:  make(map[string]*models.Post),
		bySlug: make(map[string]*models.Post),
	}
}

func (m *mockPostRepo) Create(ctx context.Context, post *models.Post) error {
	if post.ID == "" {
		post.ID = utils.NewULID()
	}
	m.posts[post.ID] = post
	m.bySlug[post.Slug] = post
	return nil
}

func (m *mockPostRepo) Update(ctx context.Context, post *models.Post) error {
	if old, ok := m.posts[post.ID]; ok && old.Slug != post.Slug {
		delete(m.bySlug, old.Slug)
	}
	m.posts[post.ID] = post
	m.bySlug[post.Slug] = post
	return nil
}

func (m *mockPostRepo) Delete(ctx context.Context, id string) error {
	if p, ok := m.posts[id]; ok {
		delete(m.bySlug, p.Slug)
		delete(m.posts, id)
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockPostRepo) GetByID(ctx context.Context, id string) (*models.Post, error) {
	if p, ok := m.posts[id]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockPostRepo) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	if p, ok := m.bySlug[slug]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockPostRepo) GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error) {
	m.getPublishedCalls++
	if p, ok := m.bySlug[slug]; ok && p.Status == models.StatusPublished {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockPostRepo) ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	var list []models.Post
	for _, p := range m.posts {
		if p.Status == models.StatusPublished {
			list = append(list, *p)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockPostRepo) ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	var list []models.Post
	for _, p := range m.posts {
		list = append(list, *p)
	}
	return list, int64(len(list)), nil
}

func (m *mockPostRepo) IncrementViews(ctx context.Context, postID string) error {
	if p, ok := m.posts[postID]; ok {
		p.Views++
	}
	return nil
}

func (m *mockPostRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.posts)), nil
}

// mockCategoryRepo implements repository.CategoryRepository in memory
type mockCategoryRepo struct {
	categories map[string]*models.Category
}

func (m *mockCategoryRepo) List(ctx context.Context) ([]models.Category, error) { return nil, nil }
func (m *mockCategoryRepo) GetByID(ctx context.Context, id string) (*models.Category, error) {
	if m.categories != nil {
		if c, ok := m.categories[id]; ok {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (m *mockCategoryRepo) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	return nil, repository.ErrNotFound
}
func (m *mockCategoryRepo) Create(ctx context.Context, cat *models.Category) error { return nil }
func (m *mockCategoryRepo) Update(ctx context.Context, cat *models.Category) error { return nil }
func (m *mockCategoryRepo) Delete(ctx context.Context, id string) error             { return nil }
func (m *mockCategoryRepo) Count(ctx context.Context) (int64, error)                { return 0, nil }

type mockMarkdownRenderer struct {
	rendered string
}

func (m *mockMarkdownRenderer) RenderToHTML(source string) (string, error) {
	return m.rendered, nil
}

func TestPostService_PublicationRules(t *testing.T) {
	postRepo := newMockPostRepo()
	catRepo := &mockCategoryRepo{}
	mdSvc := NewMarkdownService()
	svc := NewPostService(postRepo, catRepo, mdSvc)
	ctx := context.Background()

	// 1. New Post defaults to draft and published_at is nil
	createDraftInput := models.CreatePostInput{
		Type:    models.TypeArticle,
		Title:   "Draft First Post",
		Content: "Konten awal masih berupa draft.",
		Status:  models.StatusDraft,
		Tags:    []string{"Go", "go", "  MySQL  ", "Go"},
	}

	post, err := svc.Create(ctx, createDraftInput)
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	if post.Status != models.StatusDraft {
		t.Errorf("expected status 'draft', got %q", post.Status)
	}
	if post.PublishedAt != nil {
		t.Errorf("expected published_at to be nil for draft, got %v", post.PublishedAt)
	}
	if post.Slug != "draft-first-post" {
		t.Errorf("expected slug 'draft-first-post', got %q", post.Slug)
	}

	// Verify tag deduplication
	if len(post.Tags) != 2 {
		t.Errorf("expected 2 deduplicated tags, got %v", post.Tags)
	}

	// 2. Publish the post: published_at must be populated with current time
	publishInput := models.UpdatePostInput{
		Type:    models.TypeArticle,
		Title:   "Draft First Post",
		Content: "Konten sudah siap rilis.",
		Status:  models.StatusPublished,
		Tags:    post.Tags,
	}

	publishedPost, err := svc.Update(ctx, post.ID, publishInput)
	if err != nil {
		t.Fatalf("failed to publish post: %v", err)
	}

	if publishedPost.Status != models.StatusPublished {
		t.Errorf("expected status 'published', got %q", publishedPost.Status)
	}
	if publishedPost.PublishedAt == nil {
		t.Fatal("expected published_at to be set when publishing for first time")
	}
	initialPublishedAt := *publishedPost.PublishedAt

	// 3. Update while keeping published: published_at must NOT change
	time.Sleep(10 * time.Millisecond)
	updateInput := models.UpdatePostInput{
		Type:    models.TypeArticle,
		Title:   "Draft First Post Edited",
		Content: "Perbaikan kecil setelah terbit.",
		Status:  models.StatusPublished,
		Tags:    publishedPost.Tags,
	}

	updatedPost, err := svc.Update(ctx, post.ID, updateInput)
	if err != nil {
		t.Fatalf("failed to update post: %v", err)
	}

	if !updatedPost.PublishedAt.Equal(initialPublishedAt) {
		t.Errorf("published_at changed! initial: %v, current: %v", initialPublishedAt, *updatedPost.PublishedAt)
	}

	// 4. Revert to draft: published_at MUST be preserved, but public view must return 404
	revertInput := models.UpdatePostInput{
		Type:    models.TypeArticle,
		Title:   "Draft First Post Edited",
		Content: "Ditarik sementara ke draft.",
		Status:  models.StatusDraft,
		Tags:    publishedPost.Tags,
	}

	revertedPost, err := svc.Update(ctx, post.ID, revertInput)
	if err != nil {
		t.Fatalf("failed to revert post to draft: %v", err)
	}

	if revertedPost.Status != models.StatusDraft {
		t.Errorf("expected status 'draft', got %q", revertedPost.Status)
	}
	if revertedPost.PublishedAt == nil || !revertedPost.PublishedAt.Equal(initialPublishedAt) {
		t.Errorf("published_at must be preserved when reverted to draft, got: %v", revertedPost.PublishedAt)
	}

	// Public access to draft must return ErrPostNotFound (FR-021)
	_, err = svc.GetPublishedBySlug(ctx, revertedPost.Slug)
	if err == nil {
		t.Errorf("expected ErrPostNotFound for draft post accessed through public API, got nil")
	}
}

func TestPost_SEOFallback(t *testing.T) {
	excerpt := "Ringkasan ringkas konten."
	customTitle := "Custom SEO Title"
	customDesc := "Custom SEO Description"

	p1 := models.Post{
		Title:   "Judul Asli Post",
		Excerpt: &excerpt,
	}
	p1.ComputeSEOMetadata()

	if p1.SEOTitle != "Judul Asli Post" {
		t.Errorf("expected fallback SEO title to be post title, got %q", p1.SEOTitle)
	}
	if p1.SEODescription != excerpt {
		t.Errorf("expected fallback SEO description to be excerpt, got %q", p1.SEODescription)
	}

	p2 := models.Post{
		Title:           "Judul Asli Post",
		Excerpt:         &excerpt,
		MetaTitle:       &customTitle,
		MetaDescription: &customDesc,
	}
	p2.ComputeSEOMetadata()

	if p2.SEOTitle != customTitle {
		t.Errorf("expected SEO title to be custom meta_title, got %q", p2.SEOTitle)
	}
	if p2.SEODescription != customDesc {
		t.Errorf("expected SEO description to be custom meta_description, got %q", p2.SEODescription)
	}
}

func TestPostService_SlugCollisionAndCategoryValidation(t *testing.T) {
	postRepo := newMockPostRepo()
	cat1ID := utils.NewULID()
	catRepo := &mockCategoryRepo{
		categories: map[string]*models.Category{
			cat1ID: {ID: cat1ID, Name: "Go", Slug: "go"},
		},
	}
	svc := NewPostService(postRepo, catRepo, NewMarkdownService())
	ctx := context.Background()

	_, err := svc.Create(ctx, models.CreatePostInput{
		Type:       models.TypeArticle,
		Title:      "Unique Post",
		Content:    "Body",
		CategoryID: &cat1ID,
		Status:     models.StatusPublished,
	})
	if err != nil {
		t.Fatalf("unexpected error creating first post: %v", err)
	}

	// Duplicate slug on Create
	_, err = svc.Create(ctx, models.CreatePostInput{
		Type:    models.TypeBlog,
		Title:   "Unique Post",
		Content: "Another body",
	})
	if !errors.Is(err, ErrPostSlugExists) {
		t.Errorf("expected ErrPostSlugExists, got %v", err)
	}

	// Non-existent category on Create
	invalidCatID := "non-existent-cat-id"
	_, err = svc.Create(ctx, models.CreatePostInput{
		Type:       models.TypeBlog,
		Title:      "Second Post",
		Content:    "Body",
		CategoryID: &invalidCatID,
	})
	if !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("expected ErrCategoryNotFound, got %v", err)
	}

	// Create second post cleanly and test slug collision on Update
	second, err := svc.Create(ctx, models.CreatePostInput{
		Type:    models.TypeBlog,
		Title:   "Second Post",
		Content: "Body",
	})
	if err != nil {
		t.Fatalf("unexpected error creating second post: %v", err)
	}

	_, err = svc.Update(ctx, second.ID, models.UpdatePostInput{
		Type:    models.TypeBlog,
		Title:   "Second Post",
		Slug:    "unique-post",
		Content: "Updated body",
		Status:  models.StatusDraft,
	})
	if !errors.Is(err, ErrPostSlugExists) {
		t.Errorf("expected ErrPostSlugExists on update, got %v", err)
	}
}

func TestPostService_GetPublishedBySlugSingleQueryAndInterfaceRenderer(t *testing.T) {
	postRepo := newMockPostRepo()
	catRepo := &mockCategoryRepo{}
	mockMD := &mockMarkdownRenderer{rendered: "<p>mocked-html</p>"}
	svc := NewPostService(postRepo, catRepo, mockMD)
	ctx := context.Background()

	_, err := svc.Create(ctx, models.CreatePostInput{
		Type:    models.TypeArticle,
		Title:   "Perf Test",
		Content: "Markdown",
		Status:  models.StatusPublished,
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	post, err := svc.GetPublishedBySlug(ctx, "perf-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if postRepo.getPublishedCalls != 1 {
		t.Errorf("expected exactly 1 GetPublishedBySlug repository call, got %d", postRepo.getPublishedCalls)
	}
	if post.Views != 1 {
		t.Errorf("expected local incremented view count 1, got %d", post.Views)
	}
	if post.ContentHTML != "<p>mocked-html</p>" {
		t.Errorf("expected mocked markdown output, got %q", post.ContentHTML)
	}
}
