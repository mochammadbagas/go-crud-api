package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
	_ "golang.org/x/image/webp"
)

type MediaServiceInterface interface {
	Upload(ctx context.Context, file io.Reader, originalFilename string, mimeType string, size int64, uploadedBy string) (*models.Media, error)
	GetByID(ctx context.Context, id string) (*models.Media, error)
	List(ctx context.Context, page, perPage int) ([]models.Media, int64, error)
	Delete(ctx context.Context, id string) error
}

type MediaService struct {
	mediaRepo repository.MediaRepository
	store     storage.Storage
	cfg       *config.Config
}

func NewMediaService(mediaRepo repository.MediaRepository, store storage.Storage, cfg *config.Config) *MediaService {
	return &MediaService{
		mediaRepo: mediaRepo,
		store:     store,
		cfg:       cfg,
	}
}

func (s *MediaService) Upload(
	ctx context.Context,
	file io.Reader,
	originalFilename string,
	mimeType string,
	size int64,
	uploadedBy string,
) (*models.Media, error) {
	// Read entire file content into buffer for dimension decoding and upload
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		switch mimeType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		default:
			ext = ".bin"
		}
	}

	// Try extracting image dimensions
	var width, height *int32
	cfg, _, err := image.DecodeConfig(bytes.NewReader(buf))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		w := int32(cfg.Width)
		h := int32(cfg.Height)
		width = &w
		height = &h
	}

	mediaID := utils.NewULID()
	uniqueFilename := fmt.Sprintf("%s%s", mediaID, ext)

	fileURL, err := s.store.Upload(ctx, bytes.NewReader(buf), uniqueFilename, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}

	storageDisk := s.cfg.StorageDriver
	if storageDisk == "" {
		storageDisk = "local"
	}

	now := time.Now().UTC()
	media := &models.Media{
		ID:          mediaID,
		UploadedBy:  uploadedBy,
		Filename:    uniqueFilename,
		StorageDisk: storageDisk,
		StorageKey:  uniqueFilename,
		URL:         fileURL,
		MimeType:    mimeType,
		Size:        size,
		Width:       width,
		Height:      height,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.mediaRepo.Create(ctx, media); err != nil {
		// Clean up uploaded file on database failure
		_ = s.store.Delete(ctx, uniqueFilename)
		return nil, fmt.Errorf("failed to save media record to database: %w", err)
	}

	return media, nil
}

func (s *MediaService) GetByID(ctx context.Context, id string) (*models.Media, error) {
	return s.mediaRepo.GetByID(ctx, id)
}

func (s *MediaService) List(ctx context.Context, page, perPage int) ([]models.Media, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	return s.mediaRepo.List(ctx, limit, offset)
}

func (s *MediaService) Delete(ctx context.Context, id string) error {
	inUse, err := s.mediaRepo.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return repository.ErrMediaInUse
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.mediaRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Remove physical file from storage
	_ = s.store.Delete(ctx, media.StorageKey)
	return nil
}
