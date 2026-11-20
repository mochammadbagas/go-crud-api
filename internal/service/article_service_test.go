package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type mockArticleRepo struct {
	articlesByID map[string]*models.Article
	slugMap      map[string]*models.Article
}

func newMockArticleRepo() *mockArticleRepo {
	return &mockArticleRepo{
		articlesByID: make(map[string]*models.Article),
		slugMap:      make(map[string]*models.Article),
	}
}

func (m *mockArticleRepo) Create(ctx context.Context, article *models.Article, tagIDs, mediaIDs, relatedIDs []string) error {
	if _, ok := m.slugMap[article.Slug]; ok {
		return repository.ErrArticleSlugExists
	}
	if article.ID == "" {
		article.ID = utils.NewULID()
	}
	m.articlesByID[article.ID] = article
	m.slugMap[article.Slug] = article
	return nil
}

func (m *mockArticleRepo) Update(ctx context.Context, article *models.Article, tagIDs, mediaIDs, relatedIDs []string, userID string) error {
	existing, ok := m.articlesByID[article.ID]
	if !ok {
		return repository.ErrArticleNotFound
	}
	if existing.Slug != article.Slug {
		if _, collision := m.slugMap[article.Slug]; collision {
			return repository.ErrArticleSlugExists
		}
		delete(m.slugMap, existing.Slug)
		m.slugMap[article.Slug] = article
	}
	m.articlesByID[article.ID] = article
	return nil
}

func (m *mockArticleRepo) Delete(ctx context.Context, id string) error {
	existing, ok := m.articlesByID[id]
	if !ok {
		return repository.ErrArticleNotFound
	}
	// Option A: rename slug on soft-delete
	now := time.Now().UTC()
	existing.DeletedAt = &now
	delete(m.slugMap, existing.Slug)

	baseSlug := existing.Slug
	if len(baseSlug) > 220 {
		baseSlug = baseSlug[:220]
	}
	existing.Slug = baseSlug + "-deleted-" + strings.ToLower(utils.NewULID())
	m.slugMap[existing.Slug] = existing
	return nil
}

func (m *mockArticleRepo) GetByID(ctx context.Context, id string) (*models.Article, error) {
	if a, ok := m.articlesByID[id]; ok && a.DeletedAt == nil {
		return a, nil
	}
	return nil, repository.ErrArticleNotFound
}

func (m *mockArticleRepo) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	if a, ok := m.slugMap[slug]; ok && a.DeletedAt == nil {
		return a, nil
	}
	return nil, repository.ErrArticleNotFound
}

func (m *mockArticleRepo) GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error) {
	if a, ok := m.slugMap[slug]; ok && a.DeletedAt == nil && a.Status == models.StatusPublished {
		return a, nil
	}
	return nil, repository.ErrArticleNotFound
}

func (m *mockArticleRepo) ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	var list []models.Article
	for _, a := range m.articlesByID {
		if a.DeletedAt == nil && a.Status == models.StatusPublished {
			list = append(list, *a)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockArticleRepo) ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	var list []models.Article
	for _, a := range m.articlesByID {
		if a.DeletedAt == nil {
			list = append(list, *a)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockArticleRepo) PublishScheduled(ctx context.Context) error { return nil }
func (m *mockArticleRepo) Count(ctx context.Context) (int64, error)    { return int64(len(m.articlesByID)), nil }
func (m *mockArticleRepo) ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error) {
	return nil, nil
}
func (m *mockArticleRepo) CheckArticlePublishedExists(ctx context.Context, id string) (bool, error) {
	if a, ok := m.articlesByID[id]; ok && a.DeletedAt == nil && a.Status == models.StatusPublished {
		return true, nil
	}
	return false, nil
}

type mockCategoryRepo struct {
	catsByID map[string]*models.Category
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{catsByID: make(map[string]*models.Category)}
}

func (m *mockCategoryRepo) List(ctx context.Context) ([]models.Category, error) { return nil, nil }
func (m *mockCategoryRepo) GetByID(ctx context.Context, id string) (*models.Category, error) {
	if c, ok := m.catsByID[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (m *mockCategoryRepo) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	return nil, nil
}
func (m *mockCategoryRepo) Create(ctx context.Context, cat *models.Category) error {
	m.catsByID[cat.ID] = cat
	return nil
}
func (m *mockCategoryRepo) Update(ctx context.Context, cat *models.Category) error { return nil }
func (m *mockCategoryRepo) Delete(ctx context.Context, id string) error           { return nil }
func (m *mockCategoryRepo) Count(ctx context.Context) (int64, error)              { return 0, nil }
func (m *mockCategoryRepo) CountArticlesByCategoryID(ctx context.Context, categoryID string) (int64, error) {
	return 0, nil
}

type mockTagRepo struct{}

func (m *mockTagRepo) List(ctx context.Context) ([]models.Tag, error)                 { return nil, nil }
func (m *mockTagRepo) GetByID(ctx context.Context, id string) (*models.Tag, error)   { return nil, nil }
func (m *mockTagRepo) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) { return nil, nil }
func (m *mockTagRepo) GetByName(ctx context.Context, name string) (*models.Tag, error) { return nil, nil }
func (m *mockTagRepo) Create(ctx context.Context, tag *models.Tag) error              { return nil }
func (m *mockTagRepo) Update(ctx context.Context, tag *models.Tag) error              { return nil }
func (m *mockTagRepo) Delete(ctx context.Context, id string) error                    { return nil }
func (m *mockTagRepo) Count(ctx context.Context) (int64, error)                       { return 0, nil }
func (m *mockTagRepo) GetOrCreateTags(ctx context.Context, names []string) ([]models.Tag, error) {
	var tags []models.Tag
	for _, n := range names {
		tags = append(tags, models.Tag{ID: utils.NewULID(), Name: n, Slug: utils.GenerateSlug(n)})
	}
	return tags, nil
}

func TestArticleService_CRUDAndSoftDeleteSlugReuse(t *testing.T) {
	articleRepo := newMockArticleRepo()
	categoryRepo := newMockCategoryRepo()
	tagRepo := &mockTagRepo{}

	svc := service.NewArticleService(articleRepo, categoryRepo, tagRepo)
	ctx := context.Background()

	// 1. Create Article A with title "Belajar Golang"
	input1 := models.CreateArticleInput{
		Title:   "Belajar Golang",
		Content: "<p>Halo Dunia</p>",
		Status:  models.StatusPublished,
	}
	artA, err := svc.Create(ctx, "author-123", input1)
	if err != nil {
		t.Fatalf("failed to create article: %v", err)
	}
	if artA.Slug != "belajar-golang" {
		t.Errorf("expected slug 'belajar-golang', got %q", artA.Slug)
	}

	// 2. Duplicate slug should be rejected
	_, err = svc.Create(ctx, "author-123", input1)
	if !errors.Is(err, service.ErrArticleSlugExists) {
		t.Errorf("expected ErrArticleSlugExists, got %v", err)
	}

	// 3. Delete Article A (soft-delete with slug renaming Option A)
	if err := svc.Delete(ctx, artA.ID); err != nil {
		t.Fatalf("failed to delete article: %v", err)
	}

	// 4. Create Article B with the SAME slug "belajar-golang" -> MUST SUCCEED (BUG-01 Fix)
	artB, err := svc.Create(ctx, "author-456", input1)
	if err != nil {
		t.Fatalf("BUG-01 REGRESSION: creating article with previously deleted slug failed: %v", err)
	}
	if artB.Slug != "belajar-golang" {
		t.Errorf("expected new article B to take 'belajar-golang', got %q", artB.Slug)
	}

	// 5. Non-Latin title slug fallback (BUG-03 Fix)
	inputNonLatin := models.CreateArticleInput{
		Title:   "こんにちは 世界",
		Content: "<p>Japanese Title</p>",
		Status:  models.StatusPublished,
	}
	artC, err := svc.Create(ctx, "author-123", inputNonLatin)
	if err != nil {
		t.Fatalf("failed to create non-Latin article: %v", err)
	}
	if !strings.HasPrefix(artC.Slug, "post-") {
		t.Errorf("BUG-03 REGRESSION: expected fallback slug starting with 'post-', got %q", artC.Slug)
	}
}

