package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type PublicCategoryHandler struct {
	categorySvc service.CategoryServiceInterface
	postSvc     service.PostServiceInterface
}

func NewPublicCategoryHandler(categorySvc service.CategoryServiceInterface, postSvc service.PostServiceInterface) *PublicCategoryHandler {
	return &PublicCategoryHandler{
		categorySvc: categorySvc,
		postSvc:     postSvc,
	}
}

// List handles GET /api/v1/categories
func (h *PublicCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categorySvc.List(r.Context())
	if err != nil {
		slog.Error("gagal mengambil daftar kategori publik", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar kategori")
		return
	}
	utils.JSON(w, http.StatusOK, categories)
}

// ListPostsByCategory handles GET /api/v1/categories/{slug}/posts
func (h *PublicCategoryHandler) ListPostsByCategory(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.Error(w, http.StatusBadRequest, "slug kategori wajib disertakan")
		return
	}

	page, limit := utils.ParsePagination(r, 10, 50)
	q := r.URL.Query()

	postType := strings.ToLower(strings.TrimSpace(q.Get("type")))
	if err := models.ValidateType(postType); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	filter := models.PostFilter{
		CategorySlug: slug,
		Type:         postType,
		Tag:          q.Get("tag"),
		Page:         page,
		Limit:        limit,
	}

	if q.Get("featured") != "" {
		f := q.Get("featured") == "true" || q.Get("featured") == "1"
		filter.Featured = &f
	}

	posts, total, err := h.postSvc.ListPublic(r.Context(), filter)
	if err != nil {
		slog.Error("gagal mengambil konten berdasarkan kategori", slog.String("slug", slug), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil konten kategori")
		return
	}

	pagination := utils.BuildPagination(page, limit, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       posts,
		Pagination: pagination,
	})
}

