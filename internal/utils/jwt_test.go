package utils

import (
	"testing"
	"time"
)

func TestJWT_AccessTokenGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key-12345"
	ttl := 15 * time.Minute
	userID := NewULID()
	username := "bagas"
	email := "bagas@example.com"
	role := "admin"

	tokenStr, err := GenerateAccessToken(secret, ttl, userID, username, email, role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := ValidateAccessToken(secret, tokenStr)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %q, got %q", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected username %q, got %q", username, claims.Username)
	}
	if claims.Email != email {
		t.Errorf("expected email %q, got %q", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("expected role %q, got %q", role, claims.Role)
	}

	// Test invalid secret
	_, err = ValidateAccessToken("wrong-secret", tokenStr)
	if err == nil {
		t.Error("expected error when validating with wrong secret, got nil")
	}

	// Test expired token
	expiredTokenStr, err := GenerateAccessToken(secret, -1*time.Minute, userID, username, email, role)
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}
	_, err = ValidateAccessToken(secret, expiredTokenStr)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestULID_GenerationAndValidation(t *testing.T) {
	id := NewULID()
	if len(id) != 26 {
		t.Errorf("expected ULID length 26, got %d (%q)", len(id), id)
	}
	if !IsValidULID(id) {
		t.Errorf("expected %q to be valid ULID", id)
	}

	if IsValidULID("invalid-ulid-string") {
		t.Error("expected invalid ULID to return false")
	}
}

func TestToken_Hash(t *testing.T) {
	raw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}
	if len(raw) != 64 {
		t.Errorf("expected 64 chars hex token, got %d", len(raw))
	}

	h1 := HashToken(raw)
	h2 := HashToken(raw)
	if h1 != h2 {
		t.Errorf("expected identical hashes for same token, got %q and %q", h1, h2)
	}
}
