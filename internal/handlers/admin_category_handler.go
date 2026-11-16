package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type AdminCategoryHandler struct {
	categorySvc service.CategoryServiceInterface
}

func NewAdminCategoryHandler(categorySvc service.CategoryServiceInterface) *AdminCategoryHandler {
	return &AdminCategoryHandler{categorySvc: categorySvc}
}

// List handles GET /api/v1/admin/categories
func (h *AdminCategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categorySvc.List(r.Context())
	if err != nil {
		slog.Error("failed to fetch categories", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to fetch categories")
		return
	}
	utils.JSON(w, http.StatusOK, categories)
}

// Create handles POST /api/v1/admin/categories
func (h *AdminCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	cat, err := h.categorySvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrCategorySlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "CATEGORY_SLUG_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, cat)
}

// Update handles PUT /api/v1/admin/categories/{id}
func (h *AdminCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	var input models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	cat, err := h.categorySvc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, service.ErrCategorySlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "CATEGORY_SLUG_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, cat)
}

// Delete handles DELETE /api/v1/admin/categories/{id}
func (h *AdminCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	if err := h.categorySvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to delete category", slog.String("id", id), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
