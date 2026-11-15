package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Category represents a content category.
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CategoryInput is the payload for creating or updating a category.
type CategoryInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Validate validates CategoryInput.
func (i CategoryInput) Validate() error {
	v := utils.ValidationErrors{}
	if strings.TrimSpace(i.Name) == "" {
		v.Add("name", "category name is required")
	}
	if v.HasErrors() {
		return v
	}
	return nil
}
