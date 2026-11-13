package models

import (
	"errors"
	"strings"
	"time"
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
	ID              uint64     `json:"id"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Excerpt         *string    `json:"excerpt"`
	Content         string     `json:"content"`
	ContentHTML     string     `json:"content_html,omitempty"`
	CoverImage      *string    `json:"cover_image"`
	CategoryID      *uint64    `json:"category_id"`
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

// ComputeSEOMetadata applies SEO fallback rules (FR-018, FR-019, FR-020).
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
	CategoryID      *uint64  `json:"category_id"`
	Tags            []string `json:"tags"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      bool     `json:"is_featured"`
	Status          string   `json:"status"`
}

// Validate validates CreatePostInput business rules.
func (i *CreatePostInput) Validate() error {
	i.Type = strings.ToLower(strings.TrimSpace(i.Type))
	if i.Type != TypeBlog && i.Type != TypeArticle && i.Type != TypeJournal {
		return errors.New("type harus salah satu dari: blog, article, journal")
	}

	i.Title = strings.TrimSpace(i.Title)
	if i.Title == "" {
		return errors.New("title wajib diisi")
	}

	i.Content = strings.TrimSpace(i.Content)
	if i.Content == "" {
		return errors.New("content wajib diisi")
	}

	if i.Status == "" {
		i.Status = StatusDraft
	} else {
		i.Status = strings.ToLower(strings.TrimSpace(i.Status))
		if i.Status != StatusDraft && i.Status != StatusPublished {
			return errors.New("status harus salah satu dari: draft, published")
		}
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
	CategoryID      *uint64  `json:"category_id"`
	Tags            []string `json:"tags"`
	MetaTitle       *string  `json:"meta_title"`
	MetaDescription *string  `json:"meta_description"`
	IsFeatured      bool     `json:"is_featured"`
	Status          string   `json:"status"`
}

// Validate validates UpdatePostInput business rules.
func (i *UpdatePostInput) Validate() error {
	i.Type = strings.ToLower(strings.TrimSpace(i.Type))
	if i.Type != TypeBlog && i.Type != TypeArticle && i.Type != TypeJournal {
		return errors.New("type harus salah satu dari: blog, article, journal")
	}

	i.Title = strings.TrimSpace(i.Title)
	if i.Title == "" {
		return errors.New("title wajib diisi")
	}

	i.Content = strings.TrimSpace(i.Content)
	if i.Content == "" {
		return errors.New("content wajib diisi")
	}

	i.Status = strings.ToLower(strings.TrimSpace(i.Status))
	if i.Status != StatusDraft && i.Status != StatusPublished {
		return errors.New("status harus salah satu dari: draft, published")
	}

	return nil
}

// PostFilter contains criteria for querying posts.
type PostFilter struct {
	Type         string
	Tag          string
	Featured     *bool
	CategoryID   *uint64
	CategorySlug string
	Status       string
	Page         int
	Limit        int
}

// ValidateType returns an error if the Type field is non-empty and not a recognized post type.
// Handlers should call this to avoid duplicating validation logic.
func ValidateType(postType string) error {
	if postType == "" {
		return nil
	}
	if postType != TypeBlog && postType != TypeArticle && postType != TypeJournal {
		return errors.New("type tidak valid, harus salah satu dari: blog, article, journal")
	}
	return nil
}

// ValidateStatus returns an error if the Status field is non-empty and not a recognized post status.
func ValidateStatus(status string) error {
	if status == "" {
		return nil
	}
	if status != StatusDraft && status != StatusPublished {
		return errors.New("status tidak valid, harus salah satu dari: draft, published")
	}
	return nil
}

