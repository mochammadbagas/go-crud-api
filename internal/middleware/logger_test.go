package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/logger"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestLoggerMiddleware_SuccessRequest(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		AppEnv:    "development",
		LogLevel:  "debug",
		LogFormat: "json",
	}
	slog.SetDefault(logger.New(cfg, buf))

	handler := middleware.Logger(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/articles", nil)
	ctx := context.WithValue(req.Context(), chimiddleware.RequestIDKey, "test-req-id-999")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	output := buf.String()
	if !strings.Contains(output, `"path":"/api/v1/articles"`) {
		t.Fatalf("expected log to contain path, got: %s", output)
	}
	if !strings.Contains(output, `"status":200`) {
		t.Fatalf("expected log to contain status 200, got: %s", output)
	}
	if !strings.Contains(output, `"request_id":"test-req-id-999"`) {
		t.Fatalf("expected log to contain request_id, got: %s", output)
	}
}

func TestLoggerMiddleware_NoiseReduction_SkipHealthCheckOnOK(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		AppEnv:    "production",
		LogLevel:  "info",
		LogFormat: "json",
	}
	slog.SetDefault(logger.New(cfg, buf))

	handler := middleware.Logger(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("healthy"))
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	output := buf.String()
	if strings.Contains(output, `"/health"`) {
		t.Fatalf("health check 200 OK should be skipped for noise reduction, got: %s", output)
	}
}

func TestLoggerMiddleware_HealthCheck_LoggedOnFailure(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		AppEnv:    "production",
		LogLevel:  "info",
		LogFormat: "json",
	}
	slog.SetDefault(logger.New(cfg, buf))

	handler := middleware.Logger(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unhealthy"))
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	output := buf.String()
	if !strings.Contains(output, `"/health"`) || !strings.Contains(output, `"status":503`) {
		t.Fatalf("health check failure (>=500) must be logged, got: %s", output)
	}
}

func TestLoggerMiddleware_SensitiveHeaderMasking(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		AppEnv:    "production",
		LogLevel:  "info",
		LogFormat: "json",
	}
	slog.SetDefault(logger.New(cfg, buf))

	handler := middleware.Logger(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/auth/me", nil)
	rawToken := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummysecretpayloadtoken.signature"
	req.Header.Set("Authorization", rawToken)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	output := buf.String()
	if strings.Contains(output, rawToken) {
		t.Fatalf("raw Bearer token must not appear in production log! Got: %s", output)
	}
	if !strings.Contains(output, "Bearer eyJhbG...ture") {
		t.Fatalf("masked token should appear with prefix and suffix, got: %s", output)
	}
}
