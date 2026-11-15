package storage

import (
	"context"
	"io"
	"net/url"
	"path"
	"strings"
)

// Storage defines the contract for pluggable file storage providers (Local, S3, Cloudflare R2, MinIO).
type Storage interface {
	// Upload stores a file stream under the given filename and returns its accessible public URL.
	Upload(ctx context.Context, file io.Reader, filename string, contentType string) (fileURL string, err error)
	// Delete removes a file given its filename or public URL.
	Delete(ctx context.Context, fileURLOrKey string) error
}

// ExtractKey extracts the filename/key from a public URL or relative path, stripping directories and query parameters.
func ExtractKey(fileURLOrKey string) string {
	fileURLOrKey = strings.TrimSpace(fileURLOrKey)
	if fileURLOrKey == "" {
		return ""
	}

	// Normalize Windows backslashes if present
	fileURLOrKey = strings.ReplaceAll(fileURLOrKey, "\\", "/")

	if parsed, err := url.Parse(fileURLOrKey); err == nil && parsed.Path != "" {
		fileURLOrKey = parsed.Path
	}

	fileURLOrKey = strings.TrimPrefix(fileURLOrKey, "/uploads/")
	fileURLOrKey = strings.TrimPrefix(fileURLOrKey, "uploads/")

	clean := path.Base(fileURLOrKey)
	if clean == "." || clean == "/" {
		return ""
	}
	return clean
}
