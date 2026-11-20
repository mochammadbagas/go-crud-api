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
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected LogLevel 'debug' in dev mode, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Fatalf("expected LogFormat 'text' in dev mode, got %s", cfg.LogFormat)
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
	if cfg.LogLevel != "info" {
		t.Fatalf("expected LogLevel 'info' in prod mode, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("expected LogFormat 'json' in prod mode, got %s", cfg.LogFormat)
	}

	// Cleanup
	_ = os.Setenv("APP_ENV", "development")
	_ = os.Unsetenv("JWT_SECRET")
}
