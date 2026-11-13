package models

import (
	"errors"
	"strings"
	"time"
)

// User represents the administrator user entity.
type User struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LoginInput is the payload sent during administrator login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate validates the login payload.
func (i LoginInput) Validate() error {
	if strings.TrimSpace(i.Email) == "" {
		return errors.New("email wajib diisi")
	}
	if strings.TrimSpace(i.Password) == "" {
		return errors.New("password wajib diisi")
	}
	return nil
}

// UserProfile is the safe public representation of the administrator.
type UserProfile struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
