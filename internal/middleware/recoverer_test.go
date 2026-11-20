package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/logger"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRecoverer_CatchesPanicAndLogsStructured(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		AppEnv:    "production",
		LogLevel:  "info",
		LogFormat: "json",
	}
	slog.SetDefault(logger.New(cfg, buf))

	handler := middleware.Recoverer()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("database connection pointer is nil")
	}))

	req := httptest.NewRequest("GET", "/api/v1/panic-endpoint", nil)
	ctx := context.WithValue(req.Context(), chimiddleware.RequestIDKey, "panic-req-id-777")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify HTTP response
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status 500, got %d", rr.Code)
	}

	var errResp utils.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("expected JSON error response, unmarshal failed: %v", err)
	}
	if errResp.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("expected error code INTERNAL_SERVER_ERROR, got %s", errResp.Code)
	}

	// Verify structured log output
	logOutput := buf.String()
	if !strings.Contains(logOutput, "HTTP request panic recovered") {
		t.Fatalf("expected log to indicate panic recovered, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "database connection pointer is nil") {
		t.Fatalf("expected panic error message in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"request_id":"panic-req-id-777"`) {
		t.Fatalf("expected request_id in panic log, got: %s", logOutput)
	}
}
