package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Category represents a content category.
type Category struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  *string   `json:"description,omitempty"`
	ArticleCount int64     `json:"article_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CategoryInput is the payload for creating or updating a category.
type CategoryInput struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
}

// Validate validates CategoryInput.
func (i CategoryInput) Validate() error {
	v := utils.ValidationErrors{}

	name := strings.TrimSpace(i.Name)
	if name == "" {
		v.Add("name", utils.ErrCodeRequired, "category name is required")
	} else if len(name) < 2 {
		v.Add("name", utils.ErrCodeTooShort, "category name must be at least 2 characters long")
	} else if len(name) > 100 {
		v.Add("name", utils.ErrCodeTooLong, "category name must not exceed 100 characters")
	}

	slug := strings.TrimSpace(i.Slug)
	if slug != "" {
		if len(slug) > 100 {
			v.Add("slug", utils.ErrCodeTooLong, "slug must not exceed 100 characters")
		} else if !utils.IsValidSlug(slug) {
			v.Add("slug", utils.ErrCodeInvalidFormat, "slug must contain only lowercase alphanumeric characters and hyphens")
		}
	}

	if i.Description != nil && len(*i.Description) > 1000 {
		v.Add("description", utils.ErrCodeTooLong, "description must not exceed 1000 characters")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}
