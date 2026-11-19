package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type PublicCategoryHandler struct {
	categorySvc service.CategoryServiceInterface
	articleSvc  service.ArticleServiceInterface
}

func NewPublicCategoryHandler(categorySvc service.CategoryServiceInterface, articleSvc service.ArticleServiceInterface) *PublicCategoryHandler {
	return &PublicCategoryHandler{
		categorySvc: categorySvc,
		articleSvc:  articleSvc,
	}
}

// List handles GET /api/v1/categories
func (h *PublicCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categorySvc.List(r.Context())
	if err != nil {
		slog.Error("failed to fetch public categories", slog.Any("error", err))
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch categories")
		return
	}
	utils.JSON(w, http.StatusOK, categories)
}

// ListArticlesByCategory handles GET /api/v1/categories/{slug}/articles
func (h *PublicCategoryHandler) ListArticlesByCategory(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_SLUG", "category slug is required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	filter := models.ArticleFilter{
		CategorySlug: slug,
		TagSlug:      r.URL.Query().Get("tag"),
		Search:       r.URL.Query().Get("search"),
		Page:         page,
		PerPage:      perPage,
	}

	if featuredStr := r.URL.Query().Get("is_featured"); featuredStr != "" {
		isFeatured := featuredStr == "true" || featuredStr == "1"
		filter.IsFeatured = &isFeatured
	}

	articles, total, err := h.articleSvc.ListPublic(r.Context(), filter)
	if err != nil {
		slog.Error("failed to fetch articles by category", slog.String("slug", slug), slog.Any("error", err))
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch category articles")
		return
	}

	pagination := utils.BuildPagination(page, perPage, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       articles,
		Pagination: pagination,
	})
}
