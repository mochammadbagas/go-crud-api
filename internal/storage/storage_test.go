package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
)

func TestLocalStorage_UploadAndDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := storage.NewLocalStorage(tempDir)

	content := []byte("fake image content")
	filename := "test_image.png"

	// 1. Upload
	url, err := store.Upload(context.Background(), bytes.NewReader(content), filename, "image/png")
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	expectedURL := "/uploads/" + filename
	if url != expectedURL {
		t.Errorf("expected url %s, got %s", expectedURL, url)
	}

	// Verify file exists on disk
	filePath := filepath.Join(tempDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("expected content %s, got %s", string(content), string(data))
	}

	// 2. Delete
	if err := store.Delete(context.Background(), url); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	// Verify file deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, but still exists")
	}
}

func TestLocalStorage_PathTraversalProtection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := storage.NewLocalStorage(tempDir)

	maliciousFilename := "../../evil.png"
	url, err := store.Upload(context.Background(), bytes.NewReader([]byte("evil")), maliciousFilename, "image/png")
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	// Should be sanitized to just evil.png
	if url != "/uploads/evil.png" {
		t.Errorf("expected /uploads/evil.png, got %s", url)
	}

	// Verify it was saved inside tempDir and NOT outside
	expectedPath := filepath.Join(tempDir, "evil.png")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("file should have been placed safely inside tempDir")
	}
}

func TestFactory_DriverSelection(t *testing.T) {
	// 1. Default local
	cfgLocal := &config.Config{
		StorageDriver: "local",
		UploadDir:     "./test_uploads",
	}
	sLocal := storage.New(cfgLocal)
	if sLocal == nil {
		t.Fatal("expected non-nil storage")
	}

	// 2. R2/S3 without credentials should fallback to LocalStorage
	cfgIncomplete := &config.Config{
		StorageDriver: "r2",
		UploadDir:     "./test_uploads",
		S3Bucket:      "",
	}
	sFallback := storage.New(cfgIncomplete)
	if sFallback == nil {
		t.Fatal("expected non-nil storage on fallback")
	}

	// 3. R2/S3 with credentials
	cfgS3 := &config.Config{
		StorageDriver:     "s3",
		S3Bucket:          "my-bucket",
		S3Region:          "auto",
		S3Endpoint:        "https://abc12345.r2.cloudflarestorage.com",
		S3AccessKeyID:     "fake_key",
		S3SecretAccessKey: "fake_secret",
		S3PublicURL:       "https://pub-abc.r2.dev",
	}
	sS3 := storage.New(cfgS3)
	if sS3 == nil {
		t.Fatal("expected non-nil S3 storage")
	}
}

func TestExtractKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"avatar_01KAH.png", "avatar_01KAH.png"},
		{"/uploads/avatar_01KAH.png", "avatar_01KAH.png"},
		{"uploads/avatar_01KAH.png", "avatar_01KAH.png"},
		{"https://pub-12345.r2.dev/avatar_01KAH.png", "avatar_01KAH.png"},
		{"https://pub-12345.r2.dev/avatar_01KAH.png?v=2", "avatar_01KAH.png"},
		{"https://media.mydomain.com/uploads/photo.jpg?token=abc", "photo.jpg"},
		{"uploads\\subfolder\\avatar.png", "avatar.png"},
		{"http://localhost:8080/uploads/test.webp", "test.webp"},
	}

	for _, tt := range tests {
		got := storage.ExtractKey(tt.input)
		if got != tt.expected {
			t.Errorf("ExtractKey(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestLocalStorage_ExternalURLIgnored(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_uploads_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := storage.NewLocalStorage(tempDir)

	// External URLs should not cause errors and should not touch disk
	err = store.Delete(context.Background(), "https://images.unsplash.com/photo-123.jpg")
	if err != nil {
		t.Errorf("expected no error for external URL delete, got %v", err)
	}

	err = store.Delete(context.Background(), "")
	if err != nil {
		t.Errorf("expected no error for empty URL delete, got %v", err)
	}
}
