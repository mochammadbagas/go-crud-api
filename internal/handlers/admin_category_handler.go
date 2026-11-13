package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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
		slog.Error("gagal mengambil daftar kategori", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar kategori")
		return
	}
	utils.JSON(w, http.StatusOK, categories)
}

// Create handles POST /api/v1/admin/categories
func (h *AdminCategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	cat, err := h.categorySvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrCategorySlugExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, cat)
}

// Update handles PUT /api/v1/admin/categories/{id}
func (h *AdminCategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	idVal, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || idVal == 0 {
		utils.Error(w, http.StatusBadRequest, "id tidak valid, harus angka positif")
		return
	}

	var input models.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	cat, err := h.categorySvc.Update(r.Context(), idVal, input)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrCategorySlugExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, cat)
}

// Delete handles DELETE /api/v1/admin/categories/{id}
func (h *AdminCategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idVal, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || idVal == 0 {
		utils.Error(w, http.StatusBadRequest, "id tidak valid, harus angka positif")
		return
	}

	if err := h.categorySvc.Delete(r.Context(), idVal); err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal menghapus kategori", slog.String("id", chi.URLParam(r, "id")), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus kategori")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

