package storage

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
)

// New initializes the appropriate Storage provider based on configuration.
// If driver is 's3' or 'r2' but credentials are incomplete, it gracefully falls back to LocalStorage.
func New(cfg *config.Config) Storage {
	driver := strings.ToLower(strings.TrimSpace(cfg.StorageDriver))

	if driver == "s3" || driver == "r2" {
		if cfg.S3Bucket == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" {
			slog.Warn("STORAGE_DRIVER is set to 's3'/'r2' but S3_BUCKET / S3_ACCESS_KEY_ID / S3_SECRET_ACCESS_KEY are incomplete. Falling back to LocalStorage.")
			return NewLocalStorage(cfg.UploadDir)
		}

		s3Storage, err := NewS3Storage(context.Background(), S3Config{
			Bucket:          cfg.S3Bucket,
			Region:          cfg.S3Region,
			Endpoint:        cfg.S3Endpoint,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			PublicURL:       cfg.S3PublicURL,
		})
		if err != nil {
			slog.Error("Failed to initialize S3/R2 Storage, falling back to LocalStorage", slog.Any("error", err))
			return NewLocalStorage(cfg.UploadDir)
		}

		providerName := "S3 Storage"
		if driver == "r2" || strings.Contains(cfg.S3Endpoint, "r2.cloudflarestorage.com") {
			providerName = "Cloudflare R2 Storage"
		}
		slog.Info("Initializing "+providerName+" (Cloud Mode)", slog.String("bucket", cfg.S3Bucket))
		return s3Storage
	}

	slog.Info("Initializing LocalStorage (Local Mode)", slog.String("dir", cfg.UploadDir))
	return NewLocalStorage(cfg.UploadDir)
}
