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
	ErrArticleNotFound   = repository.ErrArticleNotFound
	ErrArticleSlugExists = repository.ErrArticleSlugExists
)

// ArticleServiceInterface defines business actions for article management and retrieval.
type ArticleServiceInterface interface {
	Create(ctx context.Context, authorID string, input models.CreateArticleInput) (*models.Article, error)
	Update(ctx context.Context, id string, userID string, input models.UpdateArticleInput) (*models.Article, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Article, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error)
	ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error)
	ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error)
	ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error)
}

type ArticleService struct {
	articleRepo  repository.ArticleRepository
	categoryRepo repository.CategoryRepository
	tagRepo      repository.TagRepository
}

func NewArticleService(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	tagRepo repository.TagRepository,
) *ArticleService {
	return &ArticleService{
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
	}
}

func (s *ArticleService) Create(ctx context.Context, authorID string, input models.CreateArticleInput) (*models.Article, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	existing, err := s.articleRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, ErrArticleSlugExists
	}

	if input.CategoryID != nil && *input.CategoryID != "" {
		if _, err := s.categoryRepo.GetByID(ctx, *input.CategoryID); err != nil {
			return nil, ErrCategoryNotFound
		}
	}

	// 1. Sanitize HTML content to prevent XSS
	cleanHTML := utils.SanitizeHTML(input.Content)

	// 2. Extract media IDs embedded inside HTML content
	mediaIDs := utils.ExtractMediaIDsFromHTML(cleanHTML)

	// 3. Resolve tags (auto-create any new tags on the fly)
	tags, err := s.tagRepo.GetOrCreateTags(ctx, input.Tags)
	if err != nil {
		return nil, err
	}
	tagIDs := make([]string, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	// 4. Determine publication status and timestamp rules
	now := time.Now().UTC()
	var publishedAt *time.Time
	switch input.Status {
	case models.StatusPublished:
		if input.PublishedAt != nil {
			publishedAt = input.PublishedAt
		} else {
			publishedAt = &now
		}
	case models.StatusScheduled:
		publishedAt = input.ScheduledAt
	}

	robotsIndex := true
	if input.RobotsIndex != nil {
		robotsIndex = *input.RobotsIndex
	}
	robotsFollow := true
	if input.RobotsFollow != nil {
		robotsFollow = *input.RobotsFollow
	}

	article := &models.Article{
		AuthorID:        authorID,
		CategoryID:      input.CategoryID,
		FeaturedImageID: input.FeaturedImageID,
		OgImageID:       input.OgImageID,
		Title:           strings.TrimSpace(input.Title),
		Slug:            slug,
		Excerpt:         input.Excerpt,
		Content:         cleanHTML,
		Status:          input.Status,
		IsFeatured:      input.IsFeatured,
		ScheduledAt:     input.ScheduledAt,
		PublishedAt:     publishedAt,
		MetaTitle:       input.MetaTitle,
		MetaDescription: input.MetaDescription,
		CanonicalURL:    input.CanonicalURL,
		RobotsIndex:     robotsIndex,
		RobotsFollow:    robotsFollow,
		Meta:            input.Meta,
	}

	if err := s.articleRepo.Create(ctx, article, tagIDs, mediaIDs, input.RelatedArticleIDs); err != nil {
		if errors.Is(err, repository.ErrArticleSlugExists) || errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrArticleSlugExists
		}
		return nil, err
	}

	return s.articleRepo.GetByID(ctx, article.ID)
}

func (s *ArticleService) Update(ctx context.Context, id string, userID string, input models.UpdateArticleInput) (*models.Article, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Title)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	if slug != existing.Slug {
		collision, err := s.articleRepo.GetBySlug(ctx, slug)
		if err == nil && collision != nil && collision.ID != id {
			return nil, ErrArticleSlugExists
		}
	}

	if input.CategoryID != nil && *input.CategoryID != "" {
		if _, err := s.categoryRepo.GetByID(ctx, *input.CategoryID); err != nil {
			return nil, ErrCategoryNotFound
		}
	}

	// 1. Sanitize HTML content
	cleanHTML := utils.SanitizeHTML(input.Content)

	// 2. Extract media IDs
	mediaIDs := utils.ExtractMediaIDsFromHTML(cleanHTML)

	// 3. Resolve tags
	tags, err := s.tagRepo.GetOrCreateTags(ctx, input.Tags)
	if err != nil {
		return nil, err
	}
	tagIDs := make([]string, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	// 4. Update fields
	existing.Title = strings.TrimSpace(input.Title)
	existing.Slug = slug
	existing.Excerpt = input.Excerpt
	existing.Content = cleanHTML
	existing.CategoryID = input.CategoryID
	existing.FeaturedImageID = input.FeaturedImageID
	existing.OgImageID = input.OgImageID
	existing.IsFeatured = input.IsFeatured
	existing.MetaTitle = input.MetaTitle
	existing.MetaDescription = input.MetaDescription
	existing.CanonicalURL = input.CanonicalURL
	if input.Meta != nil {
		existing.Meta = input.Meta
	}
	if input.RobotsIndex != nil {
		existing.RobotsIndex = *input.RobotsIndex
	}
	if input.RobotsFollow != nil {
		existing.RobotsFollow = *input.RobotsFollow
	}

	// Status transition rules
	if input.Status != "" {
		existing.Status = input.Status
	}
	if existing.Status == models.StatusPublished && existing.PublishedAt == nil {
		now := time.Now().UTC()
		existing.PublishedAt = &now
	}
	if input.ScheduledAt != nil {
		existing.ScheduledAt = input.ScheduledAt
	}
	if input.PublishedAt != nil {
		existing.PublishedAt = input.PublishedAt
	}

	if err := s.articleRepo.Update(ctx, existing, tagIDs, mediaIDs, input.RelatedArticleIDs, userID); err != nil {
		if errors.Is(err, repository.ErrArticleSlugExists) || errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrArticleSlugExists
		}
		return nil, err
	}

	return s.articleRepo.GetByID(ctx, id)
}

func (s *ArticleService) Delete(ctx context.Context, id string) error {
	_, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.articleRepo.Delete(ctx, id)
}

func (s *ArticleService) GetByID(ctx context.Context, id string) (*models.Article, error) {
	return s.articleRepo.GetByID(ctx, id)
}

func (s *ArticleService) GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error) {
	return s.articleRepo.GetPublishedBySlug(ctx, slug)
}

func (s *ArticleService) ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	return s.articleRepo.ListPublic(ctx, filter)
}

func (s *ArticleService) ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	return s.articleRepo.ListAdmin(ctx, filter)
}

func (s *ArticleService) ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error) {
	return s.articleRepo.ListRevisions(ctx, articleID)
}
