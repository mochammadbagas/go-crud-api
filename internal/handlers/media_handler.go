package handlers

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type MediaHandler struct {
	mediaSvc service.MediaServiceInterface
}

func NewMediaHandler(mediaSvc service.MediaServiceInterface) *MediaHandler {
	return &MediaHandler{
		mediaSvc: mediaSvc,
	}
}

// Upload handles POST /api/v1/admin/media
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// 10 MB maximum request stream
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.ErrorCode(w, http.StatusBadRequest, "FILE_TOO_LARGE", "file is too large (max 10MB) or invalid multipart format")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		utils.ErrorCode(w, http.StatusBadRequest, "FILE_REQUIRED", "media file is required (field 'image' or 'file')")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_FORMAT", "unsupported file format, please use JPG, PNG, WEBP, or GIF")
		return
	}

	// Sniff magic bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		utils.ErrorCode(w, http.StatusBadRequest, "READ_ERROR", "failed to read file header")
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
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_MIME", "invalid file type, only authentic image files are allowed")
		return
	}

	// Rewind file pointer
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			utils.ErrorCode(w, http.StatusInternalServerError, "SEEK_ERROR", "failed to process file stream")
			return
		}
	} else {
		utils.ErrorCode(w, http.StatusInternalServerError, "STREAM_ERROR", "file stream does not support seek")
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.ErrorCode(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	media, err := h.mediaSvc.Upload(r.Context(), file, header.Filename, mimeType, header.Size, userID)
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "UPLOAD_FAILED", err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, media)
}

// List handles GET /api/v1/admin/media
func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	medias, total, err := h.mediaSvc.List(r.Context(), page, perPage)
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "LIST_ERROR", "failed to retrieve media list")
		return
	}

	pagination := utils.BuildPagination(page, perPage, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       medias,
		Pagination: pagination,
	})
}

// Get handles GET /api/v1/admin/media/{id}
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid media ID, must be a 26-character ULID")
		return
	}

	media, err := h.mediaSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrMediaNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "media not found")
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch media")
		return
	}

	utils.JSON(w, http.StatusOK, media)
}

// Delete handles DELETE /api/v1/admin/media/{id}
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid media ID, must be a 26-character ULID")
		return
	}

	if err := h.mediaSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrMediaNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "media not found")
			return
		}
		if errors.Is(err, repository.ErrMediaInUse) {
			utils.ErrorCode(w, http.StatusConflict, "MEDIA_IN_USE", err.Error())
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "DELETE_ERROR", "failed to delete media")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "media deleted successfully",
	})
}
