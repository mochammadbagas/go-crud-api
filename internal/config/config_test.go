package config_test

import (
	"os"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
)

func TestConfigLoad_DefaultDevelopment(t *testing.T) {
	_ = os.Setenv("APP_ENV", "development")
	_ = os.Unsetenv("JWT_SECRET")

	cfg := config.Load()
	if cfg.AppEnv != "development" {
		t.Fatalf("expected AppEnv 'development', got %s", cfg.AppEnv)
	}
	if cfg.JWTSecret != config.DefaultInsecureJWTSecret {
		t.Fatalf("expected fallback JWTSecret in dev mode")
	}
}

func TestConfigLoad_ProductionCustomSecret(t *testing.T) {
	_ = os.Setenv("APP_ENV", "production")
	validSecret := "a-very-strong-and-secure-jwt-secret-with-more-than-32-chars"
	_ = os.Setenv("JWT_SECRET", validSecret)

	cfg := config.Load()
	if cfg.AppEnv != "production" {
		t.Fatalf("expected AppEnv 'production', got %s", cfg.AppEnv)
	}
	if cfg.JWTSecret != validSecret {
		t.Fatalf("expected JWTSecret to match provided value")
	}

	// Cleanup
	_ = os.Setenv("APP_ENV", "development")
	_ = os.Unsetenv("JWT_SECRET")
}
