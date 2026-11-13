package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	ErrPostNotFound   = errors.New("konten tidak ditemukan")
	ErrPostSlugExists = errors.New("slug konten sudah digunakan")
)

// PostServiceInterface defines business actions for post management and retrieval.
type PostServiceInterface interface {
	Create(ctx context.Context, input models.CreatePostInput) (*models.Post, error)
	Update(ctx context.Context, id uint64, input models.UpdatePostInput) (*models.Post, error)
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*models.Post, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error)
	ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error)
	ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error)
}

type PostService struct {
	postRepo     repository.PostRepository
	categoryRepo repository.CategoryRepository
	markdownSvc  MarkdownRenderer
}

func NewPostService(
	postRepo repository.PostRepository,
	categoryRepo repository.CategoryRepository,
	markdownSvc MarkdownRenderer,
) *PostService {
	return &PostService{
		postRepo:     postRepo,
		categoryRepo: categoryRepo,
		markdownSvc:  markdownSvc,
	}
}

// Create handles creating a new blog, article, or journal post.
func (s *PostService) Create(ctx context.Context, input models.CreatePostInput) (*models.Post, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	// Validate slug uniqueness globally
	existing, err := s.postRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, ErrPostSlugExists
	}

	// Validate category_id if provided
	if input.CategoryID != nil && *input.CategoryID > 0 {
		if _, err := s.categoryRepo.GetByID(ctx, *input.CategoryID); err != nil {
			return nil, ErrCategoryNotFound
		}
	}

	normalizedTags := normalizeTags(input.Tags)
	now := time.Now().UTC()

	post := &models.Post{
		Type:            input.Type,
		Title:           input.Title,
		Slug:            slug,
		Excerpt:         input.Excerpt,
		Content:         input.Content,
		CoverImage:      input.CoverImage,
		CategoryID:      input.CategoryID,
		Tags:            normalizedTags,
		MetaTitle:       input.MetaTitle,
		MetaDescription: input.MetaDescription,
		IsFeatured:      input.IsFeatured,
		Status:          input.Status,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Business rule 5.2: Saat pertama kali dipublikasikan, published_at diisi waktu publikasi
	if post.Status == models.StatusPublished {
		post.PublishedAt = &now
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrPostSlugExists
		}
		return nil, err
	}

	// Load complete post data
	return s.postRepo.GetByID(ctx, post.ID)
}

// Update handles modifying an existing post.
func (s *PostService) Update(ctx context.Context, id uint64, input models.UpdatePostInput) (*models.Post, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	// If slug changed, verify uniqueness
	if slug != existing.Slug {
		collision, err := s.postRepo.GetBySlug(ctx, slug)
		if err == nil && collision != nil && collision.ID != id {
			return nil, ErrPostSlugExists
		}
	}

	// Validate category_id if provided
	if input.CategoryID != nil && *input.CategoryID > 0 {
		if _, err := s.categoryRepo.GetByID(ctx, *input.CategoryID); err != nil {
			return nil, ErrCategoryNotFound
		}
	}

	normalizedTags := normalizeTags(input.Tags)
	now := time.Now().UTC()

	existing.Type = input.Type
	existing.Title = input.Title
	existing.Slug = slug
	existing.Excerpt = input.Excerpt
	existing.Content = input.Content
	existing.CoverImage = input.CoverImage
	existing.CategoryID = input.CategoryID
	existing.Tags = normalizedTags
	existing.MetaTitle = input.MetaTitle
	existing.MetaDescription = input.MetaDescription
	existing.IsFeatured = input.IsFeatured
	existing.UpdatedAt = now

	// Business rule 5.2:
	// - Saat pertama kali dipublikasikan, published_at diisi waktu publikasi
	// - Saat konten diperbarui, updated_at berubah tanpa mengubah published_at
	// - published_at dipertahankan jika konten yang pernah dipublikasikan dikembalikan menjadi draft
	if input.Status == models.StatusPublished {
		if existing.PublishedAt == nil {
			existing.PublishedAt = &now
		}
		existing.Status = models.StatusPublished
	} else {
		existing.Status = models.StatusDraft
		// Note: existing.PublishedAt is preserved
	}

	if err := s.postRepo.Update(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrPostSlugExists
		}
		return nil, err
	}

	return s.postRepo.GetByID(ctx, id)
}

// Delete handles removing a post.
func (s *PostService) Delete(ctx context.Context, id uint64) error {
	_, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return s.postRepo.Delete(ctx, id)
}

// GetByID returns post details for admin view (draft or published).
func (s *PostService) GetByID(ctx context.Context, id uint64) (*models.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	post.ComputeSEOMetadata()
	return post, nil
}

// GetPublishedBySlug handles public post view with safe markdown rendering and view count increment.
func (s *PostService) GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error) {
	post, err := s.postRepo.GetPublishedBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	// Increment view count in database
	_ = s.postRepo.IncrementViews(ctx, post.ID)

	// Reflect the incremented count locally to avoid a redundant DB roundtrip.
	post.Views++

	// Render Markdown to safe HTML (FR-005, NFR-005)
	htmlContent, err := s.markdownSvc.RenderToHTML(post.Content)
	if err == nil {
		post.ContentHTML = htmlContent
	}

	post.ComputeSEOMetadata()
	return post, nil
}

// ListPublic returns paginated published posts for public view.
func (s *PostService) ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	posts, total, err := s.postRepo.ListPublic(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	for i := range posts {
		posts[i].ComputeSEOMetadata()
	}

	return posts, total, nil
}

// ListAdmin returns paginated posts for administrator view.
func (s *PostService) ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	posts, total, err := s.postRepo.ListAdmin(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	for i := range posts {
		posts[i].ComputeSEOMetadata()
	}

	return posts, total, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; !exists {
			seen[lower] = struct{}{}
			result = append(result, trimmed)
		}
	}
	return result
}

