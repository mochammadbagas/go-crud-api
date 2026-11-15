package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
)

type RoleHandler struct {
	rbacSvc service.RBACServiceInterface
}

func NewRoleHandler(rbacSvc service.RBACServiceInterface) *RoleHandler {
	return &RoleHandler{rbacSvc: rbacSvc}
}

// ListRoles handles GET /api/v1/admin/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.rbacSvc.ListRoles(r.Context())
	if err != nil {
		slog.Error("gagal mengambil daftar role", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar role")
		return
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"data": roles,
	})
}

// GetRole handles GET /api/v1/admin/roles/{id}
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.rbacSvc.GetRoleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal mengambil detail role", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil detail role")
		return
	}
	utils.JSON(w, http.StatusOK, role)
}

// CreateRole handles POST /api/v1/admin/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var input models.CreateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	role, err := h.rbacSvc.CreateRole(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNameExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, role)
}

// UpdateRole handles PUT /api/v1/admin/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.UpdateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	role, err := h.rbacSvc.UpdateRole(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, role)
}

// DeleteRole handles DELETE /api/v1/admin/roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.rbacSvc.DeleteRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrCannotDeleteSystemRole) {
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		slog.Error("gagal menghapus role", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus role")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "role berhasil dihapus",
	})
}

// AssignPermissions handles PUT /api/v1/admin/roles/{id}/permissions
func (h *RoleHandler) AssignPermissions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.AssignPermissionsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	err := h.rbacSvc.AssignPermissionsToRole(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated role with permissions
	role, err := h.rbacSvc.GetRoleByID(r.Context(), id)
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]string{
			"message": "permission role berhasil diperbarui",
		})
		return
	}

	utils.JSON(w, http.StatusOK, role)
}

// ListPermissions handles GET /api/v1/admin/permissions
func (h *RoleHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.rbacSvc.ListPermissions(r.Context())
	if err != nil {
		slog.Error("gagal mengambil daftar permission", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil daftar permission")
		return
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"data": perms,
	})
}

// GetPermission handles GET /api/v1/admin/permissions/{id}
func (h *RoleHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	perm, err := h.rbacSvc.GetPermissionByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal mengambil detail permission", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal mengambil detail permission")
		return
	}
	utils.JSON(w, http.StatusOK, perm)
}

// CreatePermission handles POST /api/v1/admin/permissions
func (h *RoleHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var input models.CreatePermissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	perm, err := h.rbacSvc.CreatePermission(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNameExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, perm)
}

// UpdatePermission handles PUT /api/v1/admin/permissions/{id}
func (h *RoleHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.UpdatePermissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	perm, err := h.rbacSvc.UpdatePermission(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, perm)
}

// DeletePermission handles DELETE /api/v1/admin/permissions/{id}
func (h *RoleHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.rbacSvc.DeletePermission(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		slog.Error("gagal menghapus permission", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus permission")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "permission berhasil dihapus",
	})
}
