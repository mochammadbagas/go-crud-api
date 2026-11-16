package models

import (
	"errors"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Allowed post types
const (
	TypeBlog    = "blog"
	TypeArticle = "article"
	TypeJournal = "journal"
)

// Allowed post publication statuses
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Post represents a blog, article, or journal entry.
type Post struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Excerpt         *string    `json:"excerpt"`
	Content         string     `json:"content"`
	ContentHTML     string     `json:"content_html,omitempty"`
	CoverImage      *string    `json:"cover_image"`
	CategoryID      *string    `json:"category_id"`
	CategoryName    *string    `json:"category_name,omitempty"`
	CategorySlug    *string    `json:"category_slug,omitempty"`
	Tags            []string   `json:"tags"`
	MetaTitle       *string    `json:"meta_title"`
	MetaDescription *string    `json:"meta_description"`
	SEOTitle        string     `json:"seo_title"`
	SEODescription  string     `json:"seo_description"`
	IsFeatured      bool       `json:"is_featured"`
	Status          string     `json:"status"`
	PublishedAt     *time.Time `json:"published_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Views           uint64     `json:"views"`
}

// ComputeSEOMetadata applies SEO fallback rules.
func (p *Post) ComputeSEOMetadata() {
	if p.MetaTitle != nil && strings.TrimSpace(*p.MetaTitle) != "" {
		p.SEOTitle = *p.MetaTitle
	} else {
		p.SEOTitle = p.Title
	}

	if p.MetaDescription != nil && strings.TrimSpace(*p.MetaDescription) != "" {
		p.SEODescription = *p.MetaDescription
	} else if p.Excerpt != nil && strings.TrimSpace(*p.Excerpt) != "" {
		p.SEODescription = *p.Excerpt
	} else {
		p.SEODescription = ""
	}
}

// CreatePostInput represents the request payload for creating a post.
type CreatePostInput struct {
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	Excerpt         *string  `json:"excerpt"`
	Content         string   `json:"content"`
	CoverImage      *string  `json:"cover_image"`
	CategoryID      *string  `json:"category_id"`
	Tags            []string `json:"tags"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      bool     `json:"is_featured"`
	Status          string   `json:"status"`
}

// Validate validates CreatePostInput business rules.
func (i *CreatePostInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Type = strings.ToLower(strings.TrimSpace(i.Type))
	if i.Type == "" {
		v.Add("type", utils.ErrCodeRequired, "type is required")
	} else if i.Type != TypeBlog && i.Type != TypeArticle && i.Type != TypeJournal {
		v.Add("type", utils.ErrCodeInvalidValue, "type must be one of: blog, article, journal")
	}

	i.Title = strings.TrimSpace(i.Title)
	if i.Title == "" {
		v.Add("title", utils.ErrCodeRequired, "title is required")
	} else if len(i.Title) < 3 {
		v.Add("title", utils.ErrCodeTooShort, "title must be at least 3 characters long")
	} else if len(i.Title) > 255 {
		v.Add("title", utils.ErrCodeTooLong, "title must not exceed 255 characters")
	}

	i.Content = strings.TrimSpace(i.Content)
	if i.Content == "" {
		v.Add("content", utils.ErrCodeRequired, "content is required")
	}

	if i.CategoryID != nil {
		catID := strings.TrimSpace(*i.CategoryID)
		if catID != "" && !utils.IsValidULID(catID) {
			v.Add("category_id", utils.ErrCodeInvalidID, "category_id must be a valid 26-character ULID")
		}
		i.CategoryID = &catID
	}

	if i.MetaTitle != nil {
		metaTitle := strings.TrimSpace(*i.MetaTitle)
		if len(metaTitle) > 255 {
			v.Add("meta_title", utils.ErrCodeTooLong, "meta_title must not exceed 255 characters")
		}
		i.MetaTitle = &metaTitle
	}

	if i.Status == "" {
		i.Status = StatusDraft
	} else {
		i.Status = strings.ToLower(strings.TrimSpace(i.Status))
		if i.Status != StatusDraft && i.Status != StatusPublished {
			v.Add("status", utils.ErrCodeInvalidStatus, "status must be one of: draft, published")
		}
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// UpdatePostInput represents the request payload for updating a post.
type UpdatePostInput struct {
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	Excerpt         *string  `json:"excerpt"`
	Content         string   `json:"content"`
	CoverImage      *string  `json:"cover_image"`
	CategoryID      *string  `json:"category_id"`
	Tags            []string `json:"tags"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      bool     `json:"is_featured"`
	Status          string   `json:"status"`
}

// Validate validates UpdatePostInput business rules.
func (i *UpdatePostInput) Validate() error {
	v := utils.ValidationErrors{}

	i.Type = strings.ToLower(strings.TrimSpace(i.Type))
	if i.Type == "" {
		v.Add("type", utils.ErrCodeRequired, "type is required")
	} else if i.Type != TypeBlog && i.Type != TypeArticle && i.Type != TypeJournal {
		v.Add("type", utils.ErrCodeInvalidValue, "type must be one of: blog, article, journal")
	}

	i.Title = strings.TrimSpace(i.Title)
	if i.Title == "" {
		v.Add("title", utils.ErrCodeRequired, "title is required")
	} else if len(i.Title) < 3 {
		v.Add("title", utils.ErrCodeTooShort, "title must be at least 3 characters long")
	} else if len(i.Title) > 255 {
		v.Add("title", utils.ErrCodeTooLong, "title must not exceed 255 characters")
	}

	slug := strings.TrimSpace(i.Slug)
	if slug != "" {
		if len(slug) > 255 {
			v.Add("slug", utils.ErrCodeTooLong, "slug must not exceed 255 characters")
		} else if !utils.IsValidSlug(slug) {
			v.Add("slug", utils.ErrCodeInvalidFormat, "slug must contain only lowercase alphanumeric characters and hyphens")
		}
		i.Slug = slug
	}

	i.Content = strings.TrimSpace(i.Content)
	if i.Content == "" {
		v.Add("content", utils.ErrCodeRequired, "content is required")
	}

	if i.CategoryID != nil {
		catID := strings.TrimSpace(*i.CategoryID)
		if catID != "" && !utils.IsValidULID(catID) {
			v.Add("category_id", utils.ErrCodeInvalidID, "category_id must be a valid 26-character ULID")
		}
		i.CategoryID = &catID
	}

	if i.MetaTitle != nil {
		metaTitle := strings.TrimSpace(*i.MetaTitle)
		if len(metaTitle) > 255 {
			v.Add("meta_title", utils.ErrCodeTooLong, "meta_title must not exceed 255 characters")
		}
		i.MetaTitle = &metaTitle
	}

	i.Status = strings.ToLower(strings.TrimSpace(i.Status))
	if i.Status == "" {
		v.Add("status", utils.ErrCodeRequired, "status is required")
	} else if i.Status != StatusDraft && i.Status != StatusPublished {
		v.Add("status", utils.ErrCodeInvalidStatus, "status must be one of: draft, published")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// PostFilter contains criteria for querying posts.
type PostFilter struct {
	Type         string
	Tag          string
	Featured     *bool
	CategoryID   *string
	CategorySlug string
	Status       string
	Page         int
	Limit        int
}

// ValidateType returns an error if the Type field is non-empty and not a recognized post type.
func ValidateType(postType string) error {
	if postType == "" {
		return nil
	}
	if postType != TypeBlog && postType != TypeArticle && postType != TypeJournal {
		return errors.New("invalid type, must be one of: blog, article, journal")
	}
	return nil
}

// ValidateStatus returns an error if the Status field is non-empty and not a recognized post status.
func ValidateStatus(status string) error {
	if status == "" {
		return nil
	}
	if status != StatusDraft && status != StatusPublished {
		return errors.New("invalid status, must be one of: draft, published")
	}
	return nil
}
