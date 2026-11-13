package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type PublicPostHandler struct {
	postSvc service.PostServiceInterface
}

func NewPublicPostHandler(postSvc service.PostServiceInterface) *PublicPostHandler {
	return &PublicPostHandler{postSvc: postSvc}
}

// List handles GET /api/v1/posts
func (h *PublicPostHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.ParsePagination(r, 10, 50)
	q := r.URL.Query()

	postType := strings.ToLower(strings.TrimSpace(q.Get("type")))
	if err := models.ValidateType(postType); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	filter := models.PostFilter{
		Type:  postType,
		Tag:   q.Get("tag"),
		Page:  page,
		Limit: limit,
	}

	if q.Get("featured") != "" {
		f := q.Get("featured") == "true" || q.Get("featured") == "1"
		filter.Featured = &f
	}

	posts, total, err := h.postSvc.ListPublic(r.Context(), filter)
	if err != nil {
		slog.Error("gagal mengambil daftar konten publik", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar konten")
		return
	}

	pagination := utils.BuildPagination(page, limit, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       posts,
		Pagination: pagination,
	})
}

// Get handles GET /api/v1/posts/{slug}
func (h *PublicPostHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		utils.Error(w, http.StatusBadRequest, "slug wajib disertakan")
		return
	}

	post, err := h.postSvc.GetPublishedBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			utils.Error(w, http.StatusNotFound, "konten tidak ditemukan atau belum dipublikasikan")
			return
		}
		slog.Error("gagal mengambil konten publik", slog.String("slug", slug), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil konten")
		return
	}

	utils.JSON(w, http.StatusOK, post)
}

