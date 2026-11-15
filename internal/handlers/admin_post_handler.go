package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type AdminPostHandler struct {
	postSvc service.PostServiceInterface
	storage storage.Storage
}

func NewAdminPostHandler(postSvc service.PostServiceInterface, store storage.Storage) *AdminPostHandler {
	return &AdminPostHandler{
		postSvc: postSvc,
		storage: store,
	}
}

// List handles GET /api/v1/admin/posts
func (h *AdminPostHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.ParsePagination(r, 10, 50)
	q := r.URL.Query()

	postType := strings.ToLower(strings.TrimSpace(q.Get("type")))
	if err := models.ValidateType(postType); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	postStatus := strings.ToLower(strings.TrimSpace(q.Get("status")))
	if err := models.ValidateStatus(postStatus); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	filter := models.PostFilter{
		Type:   postType,
		Tag:    q.Get("tag"),
		Status: postStatus,
		Page:   page,
		Limit:  limit,
	}

	if q.Get("featured") != "" {
		f := q.Get("featured") == "true" || q.Get("featured") == "1"
		filter.Featured = &f
	}

	if catID := strings.TrimSpace(q.Get("category_id")); catID != "" {
		filter.CategoryID = &catID
	}

	posts, total, err := h.postSvc.ListAdmin(r.Context(), filter)
	if err != nil {
		slog.Error("gagal mengambil daftar konten admin", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar konten")
		return
	}

	pagination := utils.BuildPagination(page, limit, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       posts,
		Pagination: pagination,
	})
}

// Create handles POST /api/v1/admin/posts
func (h *AdminPostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.CreatePostInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	post, err := h.postSvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrPostSlugExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, post)
}

// Get handles GET /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "id tidak valid, harus 26 karakter ULID")
		return
	}

	post, err := h.postSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal mengambil konten admin", slog.String("id", id), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil konten")
		return
	}

	utils.JSON(w, http.StatusOK, post)
}

// Update handles PUT /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "id tidak valid, harus 26 karakter ULID")
		return
	}

	var input models.UpdatePostInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	// Deteksi cover image lama jika ada sebelum update
	var oldCoverURL string
	if existingPost, err := h.postSvc.GetByID(r.Context(), id); err == nil && existingPost != nil && existingPost.CoverImage != nil {
		oldCoverURL = *existingPost.CoverImage
	}

	post, err := h.postSvc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrPostSlugExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Auto-cleanup: Hapus cover image lama dari storage jika diganti atau dikosongkan
	if oldCoverURL != "" {
		newCoverURL := ""
		if post != nil && post.CoverImage != nil {
			newCoverURL = *post.CoverImage
		}
		if newCoverURL != oldCoverURL {
			if delErr := h.storage.Delete(r.Context(), oldCoverURL); delErr != nil {
				slog.Warn("Gagal menghapus cover image lama dari storage saat update post", slog.String("old_cover", oldCoverURL), slog.Any("error", delErr))
			}
		}
	}

	utils.JSON(w, http.StatusOK, post)
}

// Delete handles DELETE /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "id tidak valid, harus 26 karakter ULID")
		return
	}

	// Ambil post untuk mendapatkan URL cover image sebelum dihapus
	var coverImageURL string
	if post, err := h.postSvc.GetByID(r.Context(), id); err == nil && post != nil && post.CoverImage != nil {
		coverImageURL = *post.CoverImage
	}

	if err := h.postSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal menghapus konten", slog.String("id", id), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus konten")
		return
	}

	// Auto-cleanup: Hapus file cover image dari storage jika database berhasil dihapus
	if coverImageURL != "" {
		if delErr := h.storage.Delete(r.Context(), coverImageURL); delErr != nil {
			slog.Warn("Gagal menghapus cover image dari storage saat hapus post", slog.String("cover_image", coverImageURL), slog.Any("error", delErr))
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
