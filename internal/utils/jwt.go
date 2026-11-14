package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token tidak valid atau telah kedaluwarsa")
)

// JWTClaims represents custom claims inside the JWT Access Token.
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed JWT Access Token.
func GenerateAccessToken(secret string, ttl time.Duration, userID, username, email, role string) (string, error) {
	now := time.Now().UTC()
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateAccessToken parses and validates a JWT Access Token.
func ValidateAccessToken(secret string, tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode penandatanganan tidak valid: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRefreshToken generates a cryptographically secure random token (64 hex characters).
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken calculates the SHA-256 hash of a raw token string for safe database storage.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateTokenFamily generates a unique identifier for a token rotation lineage.
func GenerateTokenFamily() string {
	return NewULID()
}
