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

type AdminUserHandler struct {
	adminUserSvc service.AdminUserServiceInterface
	storage      storage.Storage
}

func NewAdminUserHandler(adminUserSvc service.AdminUserServiceInterface, store storage.Storage) *AdminUserHandler {
	return &AdminUserHandler{
		adminUserSvc: adminUserSvc,
		storage:      store,
	}
}

// ListUsers handles GET /api/v1/admin/users
func (h *AdminUserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.ParsePagination(r, 10, 50)
	q := r.URL.Query()

	filter := models.AdminUserFilter{
		Search: strings.TrimSpace(q.Get("search")),
		Status: strings.TrimSpace(q.Get("status")),
		Role:   strings.TrimSpace(q.Get("role")),
		Page:   page,
		Limit:  limit,
	}

	users, total, err := h.adminUserSvc.ListUsers(r.Context(), filter)
	if err != nil {
		slog.Error("failed to fetch admin users", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	pagination := utils.BuildPagination(page, limit, total)
	utils.JSON(w, http.StatusOK, utils.PaginatedResponse{
		Data:       users,
		Pagination: pagination,
	})
}

// GetUser handles GET /api/v1/admin/users/{id}
func (h *AdminUserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.adminUserSvc.GetUserDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("failed to get user detail", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to get user detail")
		return
	}
	utils.JSON(w, http.StatusOK, user)
}

// UpdateUser handles PUT /api/v1/admin/users/{id}
func (h *AdminUserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.AdminUpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.adminUserSvc.UpdateUser(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrEmailExists) || errors.Is(err, service.ErrUsernameExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

// UpdateStatus handles PUT /api/v1/admin/users/{id}/status
func (h *AdminUserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.UpdateUserStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.adminUserSvc.UpdateUserStatus(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrCannotDeactivateLastSuperAdmin) {
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

// AssignRoles handles PUT /api/v1/admin/users/{id}/roles
func (h *AdminUserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.AssignRolesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.adminUserSvc.AssignUserRoles(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, service.ErrCannotDemoteLastSuperAdmin) {
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, user)
}

// DeleteUser handles DELETE /api/v1/admin/users/{id}
func (h *AdminUserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Deteksi avatar pengguna sebelum akun dihapus dari database
	var avatarURL string
	if detail, err := h.adminUserSvc.GetUserDetail(r.Context(), id); err == nil && detail != nil && detail.AvatarURL != nil {
		avatarURL = *detail.AvatarURL
	}

	err := h.adminUserSvc.DeleteUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrCannotDeleteLastSuperAdmin) {
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		slog.Error("failed to delete user", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	// Auto-cleanup: Hapus avatar pengguna dari storage
	if avatarURL != "" {
		if delErr := h.storage.Delete(r.Context(), avatarURL); delErr != nil {
			slog.Warn("Failed to delete user avatar from storage during account removal", slog.String("avatar_url", avatarURL), slog.Any("error", delErr))
		}
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}
