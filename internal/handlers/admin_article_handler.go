package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type AdminArticleHandler struct {
	articleSvc service.ArticleServiceInterface
}

func NewAdminArticleHandler(articleSvc service.ArticleServiceInterface) *AdminArticleHandler {
	return &AdminArticleHandler{
		articleSvc: articleSvc,
	}
}

// List handles GET /api/v1/admin/articles
func (h *AdminArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := models.ArticleFilter{
		Page:         page,
		PerPage:      perPage,
		Search:       r.URL.Query().Get("search"),
		CategoryID:   r.URL.Query().Get("category_id"),
		CategorySlug: r.URL.Query().Get("category"),
		TagSlug:      r.URL.Query().Get("tag"),
		Status:       r.URL.Query().Get("status"),
		AuthorID:     r.URL.Query().Get("author_id"),
	}

	if featuredStr := r.URL.Query().Get("is_featured"); featuredStr != "" {
		isFeatured := featuredStr == "true" || featuredStr == "1"
		filter.IsFeatured = &isFeatured
	}

	articles, total, err := h.articleSvc.ListAdmin(r.Context(), filter)
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "LIST_ERROR", "failed to retrieve articles")
		return
	}

	pagination := utils.BuildPagination(page, perPage, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       articles,
		Pagination: pagination,
	})
}

// Get handles GET /api/v1/admin/articles/{id}
func (h *AdminArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID, must be a 26-character ULID")
		return
	}

	article, err := h.articleSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch article by ID", slog.String("id", id), slog.Any("error", err))
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch article")
		return
	}

	utils.JSON(w, http.StatusOK, article)
}

// Create handles POST /api/v1/admin/articles
func (h *AdminArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateArticleInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	authorID := middleware.GetUserIDFromContext(r.Context())
	if authorID == "" {
		utils.ErrorCode(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	article, err := h.articleSvc.Create(r.Context(), authorID, input)
	if err != nil {
		if errors.Is(err, service.ErrArticleSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "SLUG_EXISTS", err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", "specified category does not exist")
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, article)
}

// Update handles PUT /api/v1/admin/articles/{id}
func (h *AdminArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID, must be a 26-character ULID")
		return
	}

	var input models.UpdateArticleInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.ErrorCode(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	article, err := h.articleSvc.Update(r.Context(), id, userID, input)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		if errors.Is(err, service.ErrArticleSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "SLUG_EXISTS", err.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", "specified category does not exist")
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, article)
}

// Delete handles DELETE /api/v1/admin/articles/{id}
func (h *AdminArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID, must be a 26-character ULID")
		return
	}

	if err := h.articleSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "DELETE_ERROR", "failed to delete article")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "article deleted successfully",
	})
}

// ListRevisions handles GET /api/v1/admin/articles/{id}/revisions
func (h *AdminArticleHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID, must be a 26-character ULID")
		return
	}

	revisions, err := h.articleSvc.ListRevisions(r.Context(), id)
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "REVISION_ERROR", "failed to fetch revisions")
		return
	}

	utils.JSON(w, http.StatusOK, revisions)
}
