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
			slog.Warn("Peringatan: STORAGE_DRIVER disetel 's3'/'r2' tetapi S3_BUCKET / S3_ACCESS_KEY_ID / S3_SECRET_ACCESS_KEY belum lengkap. Fallback otomatis ke LocalStorage.")
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
			slog.Error("Gagal menginisialisasi S3/R2 Storage, fallback otomatis ke LocalStorage", slog.Any("error", err))
			return NewLocalStorage(cfg.UploadDir)
		}

		providerName := "S3 Storage"
		if driver == "r2" || strings.Contains(cfg.S3Endpoint, "r2.cloudflarestorage.com") {
			providerName = "Cloudflare R2 Storage"
		}
		slog.Info("Menginisialisasi "+providerName+" (Mode Cloud)", slog.String("bucket", cfg.S3Bucket))
		return s3Storage
	}

	slog.Info("Menginisialisasi LocalStorage (Mode Lokal)", slog.String("dir", cfg.UploadDir))
	return NewLocalStorage(cfg.UploadDir)
}
