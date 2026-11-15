package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type localStorage struct {
	uploadDir string
}

// NewLocalStorage creates a storage provider that writes files to local disk.
func NewLocalStorage(uploadDir string) Storage {
	return &localStorage{
		uploadDir: uploadDir,
	}
}

// Upload writes the file stream to the local upload directory.
func (s *localStorage) Upload(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Clean filename to prevent path traversal
	cleanName := filepath.Base(filename)

	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori upload lokal: %w", err)
	}

	dstPath := filepath.Join(s.uploadDir, cleanName)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("gagal membuat file lokal: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("gagal menulis stream file lokal: %w", err)
	}

	return "/uploads/" + cleanName, nil
}

// Delete removes the file from the local upload directory.
func (s *localStorage) Delete(ctx context.Context, fileURLOrKey string) error {
	fileURLOrKey = strings.TrimSpace(fileURLOrKey)
	if fileURLOrKey == "" {
		return nil
	}

	// If it's an external URL that doesn't contain /uploads/, skip safely
	if (strings.HasPrefix(fileURLOrKey, "http://") || strings.HasPrefix(fileURLOrKey, "https://")) && !strings.Contains(fileURLOrKey, "/uploads/") {
		return nil
	}

	cleanKey := ExtractKey(fileURLOrKey)
	if cleanKey == "" {
		return nil
	}

	dstPath := filepath.Join(s.uploadDir, cleanKey)
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("gagal menghapus file lokal: %w", err)
	}
	return nil
}
