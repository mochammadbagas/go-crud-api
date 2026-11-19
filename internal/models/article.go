package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Allowed article publication statuses
const (
	StatusDraft     = "draft"
	StatusScheduled = "scheduled"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

// ArticleSummary represents lightweight article data for relations and lists.
type ArticleSummary struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Excerpt          *string    `json:"excerpt,omitempty"`
	FeaturedImageURL *string    `json:"featured_image_url,omitempty"`
	Status           string     `json:"status,omitempty"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ArticleRevision represents an audit trail snapshot of an article.
type ArticleRevision struct {
	ID        string          `json:"id"`
	ArticleID string          `json:"article_id"`
	UserID    string          `json:"user_id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Snapshot  json.RawMessage `json:"snapshot"`
	CreatedAt time.Time       `json:"created_at"`
}

// Article represents an editorial publication entry.
type Article struct {
	ID               string          `json:"id"`
	AuthorID         string          `json:"author_id"`
	AuthorUsername   string          `json:"author_username,omitempty"`
	CategoryID       *string         `json:"category_id"`
	CategoryName     *string         `json:"category_name,omitempty"`
	CategorySlug     *string         `json:"category_slug,omitempty"`
	FeaturedImageID  *string         `json:"featured_image_id,omitempty"`
	FeaturedImageURL *string         `json:"featured_image_url,omitempty"`
	OgImageID        *string         `json:"og_image_id,omitempty"`
	OgImageURL       *string         `json:"og_image_url,omitempty"`
	Title            string          `json:"title"`
	Slug             string          `json:"slug"`
	Excerpt          *string         `json:"excerpt,omitempty"`
	Content          string          `json:"content"`
	Status           string          `json:"status"`
	IsFeatured       bool            `json:"is_featured"`
	ScheduledAt      *time.Time      `json:"scheduled_at,omitempty"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	MetaTitle        *string         `json:"meta_title,omitempty"`
	MetaDescription  *string         `json:"meta_description,omitempty"`
	SEOTitle         string          `json:"seo_title"`
	SEODescription   string          `json:"seo_description"`
	CanonicalURL     *string         `json:"canonical_url,omitempty"`
	RobotsIndex      bool            `json:"robots_index"`
	RobotsFollow     bool            `json:"robots_follow"`
	Meta             json.RawMessage `json:"meta,omitempty"`
	Tags             []Tag           `json:"tags,omitempty"`
	Media            []Media         `json:"media,omitempty"`
	RelatedArticles  []ArticleSummary `json:"related_articles,omitempty"`
	Views            uint64          `json:"views"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty"`
}

// ComputeSEOMetadata applies SEO fallback rules.
func (a *Article) ComputeSEOMetadata() {
	if a.MetaTitle != nil && strings.TrimSpace(*a.MetaTitle) != "" {
		a.SEOTitle = *a.MetaTitle
	} else {
		a.SEOTitle = a.Title
	}

	if a.MetaDescription != nil && strings.TrimSpace(*a.MetaDescription) != "" {
		a.SEODescription = *a.MetaDescription
	} else if a.Excerpt != nil && strings.TrimSpace(*a.Excerpt) != "" {
		a.SEODescription = *a.Excerpt
	} else {
		clean := strings.TrimSpace(utils.StripHTMLTags(a.Content))
		if len(clean) > 160 {
			a.SEODescription = clean[:157] + "..."
		} else {
			a.SEODescription = clean
		}
	}
}

// CreateArticleInput represents payload for creating an article.
type CreateArticleInput struct {
	Title             string          `json:"title"`
	Slug              string          `json:"slug,omitempty"`
	Excerpt           *string         `json:"excerpt,omitempty"`
	Content           string          `json:"content"`
	CategoryID        *string         `json:"category_id,omitempty"`
	FeaturedImageID   *string         `json:"featured_image_id,omitempty"`
	OgImageID         *string         `json:"og_image_id,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
	RelatedArticleIDs []string        `json:"related_article_ids,omitempty"`
	Status            string          `json:"status"`
	IsFeatured        bool            `json:"is_featured"`
	ScheduledAt       *time.Time      `json:"scheduled_at,omitempty"`
	PublishedAt       *time.Time      `json:"published_at,omitempty"`
	MetaTitle         *string         `json:"meta_title,omitempty"`
	MetaDescription   *string         `json:"meta_description,omitempty"`
	CanonicalURL      *string         `json:"canonical_url,omitempty"`
	RobotsIndex       *bool           `json:"robots_index,omitempty"`
	RobotsFollow      *bool           `json:"robots_follow,omitempty"`
	Meta              json.RawMessage `json:"meta,omitempty"`
}

// Validate validates CreateArticleInput.
func (i *CreateArticleInput) Validate() error {
	v := utils.ValidationErrors{}

	title := strings.TrimSpace(i.Title)
	if title == "" {
		v.Add("title", utils.ErrCodeRequired, "title is required")
	} else if len(title) > 255 {
		v.Add("title", utils.ErrCodeTooLong, "title must not exceed 255 characters")
	}

	if strings.TrimSpace(i.Content) == "" {
		v.Add("content", utils.ErrCodeRequired, "content is required")
	}

	slug := strings.TrimSpace(i.Slug)
	if slug != "" {
		if len(slug) > 255 {
			v.Add("slug", utils.ErrCodeTooLong, "slug must not exceed 255 characters")
		} else if !utils.IsValidSlug(slug) {
			v.Add("slug", utils.ErrCodeInvalidFormat, "slug must contain only lowercase alphanumeric characters and hyphens")
		}
	}

	status := strings.TrimSpace(i.Status)
	if status == "" {
		i.Status = StatusDraft
	} else if status != StatusDraft && status != StatusScheduled && status != StatusPublished && status != StatusArchived {
		v.Add("status", utils.ErrCodeInvalidFormat, "status must be one of: draft, scheduled, published, archived")
	}

	if status == StatusScheduled {
		if i.ScheduledAt == nil {
			v.Add("scheduled_at", utils.ErrCodeRequired, "scheduled_at date is required when status is scheduled")
		} else if i.ScheduledAt.Before(time.Now().UTC()) {
			v.Add("scheduled_at", utils.ErrCodeInvalidFormat, "scheduled_at must be in the future")
		}
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdateArticleInput represents payload for updating an article.
type UpdateArticleInput struct {
	Title             string          `json:"title"`
	Slug              string          `json:"slug,omitempty"`
	Excerpt           *string         `json:"excerpt,omitempty"`
	Content           string          `json:"content"`
	CategoryID        *string         `json:"category_id,omitempty"`
	FeaturedImageID   *string         `json:"featured_image_id,omitempty"`
	OgImageID         *string         `json:"og_image_id,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
	RelatedArticleIDs []string        `json:"related_article_ids,omitempty"`
	Status            string          `json:"status"`
	IsFeatured        bool            `json:"is_featured"`
	ScheduledAt       *time.Time      `json:"scheduled_at,omitempty"`
	PublishedAt       *time.Time      `json:"published_at,omitempty"`
	MetaTitle         *string         `json:"meta_title,omitempty"`
	MetaDescription   *string         `json:"meta_description,omitempty"`
	CanonicalURL      *string         `json:"canonical_url,omitempty"`
	RobotsIndex       *bool           `json:"robots_index,omitempty"`
	RobotsFollow      *bool           `json:"robots_follow,omitempty"`
	Meta              json.RawMessage `json:"meta,omitempty"`
}

// Validate validates UpdateArticleInput.
func (i *UpdateArticleInput) Validate() error {
	v := utils.ValidationErrors{}

	title := strings.TrimSpace(i.Title)
	if title == "" {
		v.Add("title", utils.ErrCodeRequired, "title is required")
	} else if len(title) > 255 {
		v.Add("title", utils.ErrCodeTooLong, "title must not exceed 255 characters")
	}

	if strings.TrimSpace(i.Content) == "" {
		v.Add("content", utils.ErrCodeRequired, "content is required")
	}

	slug := strings.TrimSpace(i.Slug)
	if slug != "" {
		if len(slug) > 255 {
			v.Add("slug", utils.ErrCodeTooLong, "slug must not exceed 255 characters")
		} else if !utils.IsValidSlug(slug) {
			v.Add("slug", utils.ErrCodeInvalidFormat, "slug must contain only lowercase alphanumeric characters and hyphens")
		}
	}

	status := strings.TrimSpace(i.Status)
	if status != "" && status != StatusDraft && status != StatusScheduled && status != StatusPublished && status != StatusArchived {
		v.Add("status", utils.ErrCodeInvalidFormat, "status must be one of: draft, scheduled, published, archived")
	}

	if status == StatusScheduled && i.ScheduledAt == nil {
		v.Add("scheduled_at", utils.ErrCodeRequired, "scheduled_at date is required when status is scheduled")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// ArticleFilter represents query filtering options for article listing.
type ArticleFilter struct {
	Page         int
	PerPage      int
	Search       string
	CategoryID   string
	CategorySlug string
	TagSlug      string
	Status       string
	IsFeatured   *bool
	AuthorID     string
	SortBy       string
	SortOrder    string
}
