package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type TagHandler struct {
	tagSvc service.TagServiceInterface
}

func NewTagHandler(tagSvc service.TagServiceInterface) *TagHandler {
	return &TagHandler{tagSvc: tagSvc}
}

// List handles GET /api/v1/tags and GET /api/v1/admin/tags
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.tagSvc.List(r.Context())
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "LIST_ERROR", "failed to fetch tags")
		return
	}
	utils.JSON(w, http.StatusOK, tags)
}

// Get handles GET /api/v1/admin/tags/{id}
func (h *TagHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID, must be a 26-character ULID")
		return
	}

	tag, err := h.tagSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTagNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch tag")
		return
	}

	utils.JSON(w, http.StatusOK, tag)
}

// Create handles POST /api/v1/admin/tags
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.TagInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	tag, err := h.tagSvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, repository.ErrTagSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "TAG_SLUG_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, tag)
}

// Update handles PUT /api/v1/admin/tags/{id}
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID, must be a 26-character ULID")
		return
	}

	var input models.TagInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	tag, err := h.tagSvc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrTagNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		if errors.Is(err, repository.ErrTagSlugExists) {
			utils.ErrorCode(w, http.StatusConflict, "TAG_SLUG_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, tag)
}

// Delete handles DELETE /api/v1/admin/tags/{id}
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(id) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid tag ID, must be a 26-character ULID")
		return
	}

	if err := h.tagSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrTagNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "tag not found")
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "DELETE_ERROR", "failed to delete tag")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "tag deleted successfully",
	})
}
