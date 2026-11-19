package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Tag represents a content label/tag.
type Tag struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	UsageCount int64     `json:"usage_count,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TagInput is the payload for creating or updating a tag.
type TagInput struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// Validate validates TagInput.
func (i TagInput) Validate() error {
	v := utils.ValidationErrors{}

	name := strings.TrimSpace(i.Name)
	if name == "" {
		v.Add("name", utils.ErrCodeRequired, "tag name is required")
	} else if len(name) < 2 {
		v.Add("name", utils.ErrCodeTooShort, "tag name must be at least 2 characters long")
	} else if len(name) > 100 {
		v.Add("name", utils.ErrCodeTooLong, "tag name must not exceed 100 characters")
	}

	slug := strings.TrimSpace(i.Slug)
	if slug != "" {
		if len(slug) > 100 {
			v.Add("slug", utils.ErrCodeTooLong, "slug must not exceed 100 characters")
		} else if !utils.IsValidSlug(slug) {
			v.Add("slug", utils.ErrCodeInvalidFormat, "slug must contain only lowercase alphanumeric characters and hyphens")
		}
	}

	if v.HasErrors() {
		return v
	}
	return nil
}
