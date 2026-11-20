package logger_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/logger"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestLogger_TextFormatAndLogLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		LogLevel:  "debug",
		LogFormat: "text",
	}

	l := logger.New(cfg, buf)
	l.Debug("test debug message")

	output := buf.String()
	if !strings.Contains(output, "level=DEBUG") || !strings.Contains(output, "test debug message") {
		t.Fatalf("expected debug message in text output, got: %s", output)
	}
}

func TestLogger_JSONFormatAndLogLevel(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		LogLevel:  "info",
		LogFormat: "json",
	}

	l := logger.New(cfg, buf)
	l.Debug("should be ignored")
	l.Info("should appear", "key", "value")

	output := buf.String()
	if strings.Contains(output, "should be ignored") {
		t.Fatalf("debug log should be filtered out by info level, got: %s", output)
	}
	if !strings.Contains(output, `"level":"INFO"`) || !strings.Contains(output, `"msg":"should appear"`) {
		t.Fatalf("expected valid JSON info log, got: %s", output)
	}
}

func TestLogger_ContextRequestIDPropagation(t *testing.T) {
	buf := new(bytes.Buffer)
	cfg := &config.Config{
		LogLevel:  "info",
		LogFormat: "json",
	}

	l := logger.New(cfg, buf)

	// Context with Chi RequestID
	ctx := context.WithValue(context.Background(), chimiddleware.RequestIDKey, "req-12345-test")
	l.InfoContext(ctx, "authenticated operation completed")

	output := buf.String()
	if !strings.Contains(output, `"request_id":"req-12345-test"`) {
		t.Fatalf("expected request_id in log attributes, got: %s", output)
	}
}
