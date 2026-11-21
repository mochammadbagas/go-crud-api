package database_test

import (
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/database"
)

func TestInit_SafeguardFreshInProduction(t *testing.T) {
	cfg := &config.Config{
		AppEnv: "production",
	}

	_, err := database.Init(cfg, true)
	if err == nil {
		t.Fatalf("expected error when running Init with fresh=true in production, got nil")
	}

	expectedSubstr := "cannot run migrations with -fresh flag in production environment"
	if err.Error() != "cannot run migrations with -fresh flag in production environment to prevent data loss" {
		t.Fatalf("expected error containing %q, got %q", expectedSubstr, err.Error())
	}
}
