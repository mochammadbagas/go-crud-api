package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type AdminUploadHandler struct {
	cfg     *config.Config
	storage storage.Storage
}

func NewAdminUploadHandler(cfg *config.Config, store storage.Storage) *AdminUploadHandler {
	return &AdminUploadHandler{
		cfg:     cfg,
		storage: store,
	}
}

// Upload handles POST /api/v1/admin/upload for storing cover images via pluggable storage (Local/R2/S3)
func (h *AdminUploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Enforce 10 MB maximum request stream size to prevent disk/memory exhaustion DoS
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "file is too large (max 10MB) or invalid multipart format")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "image file is required (field 'image' or 'file')")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
		utils.Error(w, http.StatusBadRequest, "unsupported file format, please use JPG, PNG, WEBP, or GIF")
		return
	}

	// Validate content via magic bytes to prevent file spoofing
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		utils.Error(w, http.StatusBadRequest, "failed to read image file header")
		return
	}

	mimeType := http.DetectContentType(buf[:n])
	allowedMIMEs := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	if !allowedMIMEs[mimeType] {
		utils.Error(w, http.StatusBadRequest, "invalid file type, only authentic image files are allowed")
		return
	}

	// Rewind file pointer after reading magic bytes
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			utils.Error(w, http.StatusInternalServerError, "failed to process file stream")
			return
		}
	} else {
		utils.Error(w, http.StatusInternalServerError, "file stream does not support seek")
		return
	}

	// Rename nama file keseluruhan menggunakan ULID murni (tidak menyisakan nama file asal demi keamanan & keunikan)
	uniqueName := fmt.Sprintf("%s%s", utils.NewULID(), ext)

	fileURL, err := h.storage.Upload(r.Context(), file, uniqueName, mimeType)
	if err != nil {
		slog.Error("failed to save uploaded file", slog.String("filename", uniqueName), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	utils.JSON(w, http.StatusCreated, map[string]string{
		"url":      fileURL,
		"filename": uniqueName,
	})
}
