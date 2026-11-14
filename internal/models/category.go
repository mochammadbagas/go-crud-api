package models

import (
	"errors"
	"strings"
	"time"
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
	if strings.TrimSpace(i.Name) == "" {
		return errors.New("nama kategori wajib diisi")
	}
	return nil
}
