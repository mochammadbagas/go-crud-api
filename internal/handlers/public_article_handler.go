package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type PublicArticleHandler struct {
	articleSvc service.ArticleServiceInterface
}

func NewPublicArticleHandler(articleSvc service.ArticleServiceInterface) *PublicArticleHandler {
	return &PublicArticleHandler{
		articleSvc: articleSvc,
	}
}

// List handles GET /api/v1/articles
func (h *PublicArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	filter := models.ArticleFilter{
		Page:         page,
		PerPage:      perPage,
		Search:       r.URL.Query().Get("search"),
		CategorySlug: r.URL.Query().Get("category"),
		TagSlug:      r.URL.Query().Get("tag"),
	}

	if featuredStr := r.URL.Query().Get("is_featured"); featuredStr != "" {
		isFeatured := featuredStr == "true" || featuredStr == "1"
		filter.IsFeatured = &isFeatured
	}

	articles, total, err := h.articleSvc.ListPublic(r.Context(), filter)
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

// Get handles GET /api/v1/articles/{slug}
func (h *PublicArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_SLUG", "article slug is required")
		return
	}

	article, err := h.articleSvc.GetPublishedBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		slog.Error("failed to fetch article by slug", slog.String("slug", slug), slog.Any("error", err))
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch article")
		return
	}

	utils.JSON(w, http.StatusOK, article)
}
