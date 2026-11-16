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
		slog.Error("failed to fetch admin posts", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to fetch posts")
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
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	post, err := h.postSvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrPostSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "POST_SLUG_EXISTS", err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, post)
}

// Get handles GET /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	post, err := h.postSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "POST_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to get admin post", slog.String("id", id), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to get post")
		return
	}

	utils.JSON(w, http.StatusOK, post)
}

// Update handles PUT /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	var input models.UpdatePostInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	// Detect old cover image if any before update
	var oldCoverURL string
	if existingPost, err := h.postSvc.GetByID(r.Context(), id); err == nil && existingPost != nil && existingPost.CoverImage != nil {
		oldCoverURL = *existingPost.CoverImage
	}

	post, err := h.postSvc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "POST_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, service.ErrPostSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "POST_SLUG_EXISTS", err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	// Auto-cleanup: Delete old cover image from storage if replaced or cleared
	if oldCoverURL != "" {
		newCoverURL := ""
		if post != nil && post.CoverImage != nil {
			newCoverURL = *post.CoverImage
		}
		if newCoverURL != oldCoverURL {
			if delErr := h.storage.Delete(r.Context(), oldCoverURL); delErr != nil {
				slog.Warn("Failed to delete old cover image from storage during post update", slog.String("old_cover", oldCoverURL), slog.Any("error", delErr))
			}
		}
	}

	utils.JSON(w, http.StatusOK, post)
}

// Delete handles DELETE /api/v1/admin/posts/{id}
func (h *AdminPostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	// Fetch post to retrieve cover image URL before deletion
	var coverImageURL string
	if post, err := h.postSvc.GetByID(r.Context(), id); err == nil && post != nil && post.CoverImage != nil {
		coverImageURL = *post.CoverImage
	}

	if err := h.postSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "POST_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to delete post", slog.String("id", id), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to delete post")
		return
	}

	// Auto-cleanup: Delete cover image file from storage if post was deleted from database
	if coverImageURL != "" {
		if delErr := h.storage.Delete(r.Context(), coverImageURL); delErr != nil {
			slog.Warn("Failed to delete cover image from storage during post deletion", slog.String("cover_image", coverImageURL), slog.Any("error", delErr))
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
