package routes_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
)

func TestSecurityHeadersInRouter(t *testing.T) {
	cfg := &config.Config{
		UploadDir: t.TempDir(),
	}

	router := routes.Setup(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expected := range expectedHeaders {
		got := rec.Header().Get(header)
		if got != expected {
			t.Errorf("expected %s: %s, got %s", header, expected, got)
		}
	}
}

func TestStaticFiles_PreventDirectoryListing(t *testing.T) {
	tempUploadDir := t.TempDir()

	// Write a test image file
	testFilePath := filepath.Join(tempUploadDir, "sample.txt")
	if err := os.WriteFile(testFilePath, []byte("hello file"), 0644); err != nil {
		t.Fatalf("failed to create sample file: %v", err)
	}

	cfg := &config.Config{
		UploadDir: tempUploadDir,
	}

	router := routes.Setup(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// 1. Direct file download should succeed
	fileReq := httptest.NewRequest(http.MethodGet, "/uploads/sample.txt", nil)
	fileRec := httptest.NewRecorder()
	router.ServeHTTP(fileRec, fileReq)

	if fileRec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 for direct file download, got %d", fileRec.Code)
	}
	if fileRec.Body.String() != "hello file" {
		t.Fatalf("expected body 'hello file', got %s", fileRec.Body.String())
	}

	// 2. Directory root listing should be 404 (prevent directory browsing)
	dirReq := httptest.NewRequest(http.MethodGet, "/uploads/", nil)
	dirRec := httptest.NewRecorder()
	router.ServeHTTP(dirRec, dirReq)

	if dirRec.Code != http.StatusNotFound {
		t.Fatalf("expected HTTP 404 when listing directory /uploads/, got %d", dirRec.Code)
	}
}
