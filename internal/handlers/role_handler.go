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
		slog.Error("failed to fetch roles", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to fetch roles")
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
			utils.ErrorCode(w, http.StatusNotFound, "ROLE_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to get role detail", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to get role detail")
		return
	}
	utils.JSON(w, http.StatusOK, role)
}

// CreateRole handles POST /api/v1/admin/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var input models.CreateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	role, err := h.rbacSvc.CreateRole(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNameExists) {
			utils.ErrorCode(w, http.StatusConflict, "ROLE_NAME_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, role)
}

// UpdateRole handles PUT /api/v1/admin/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.UpdateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	role, err := h.rbacSvc.UpdateRole(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "ROLE_NOT_FOUND", err.Error())
			return
		}
		utils.ValidationError(w, err)
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
			utils.ErrorCode(w, http.StatusNotFound, "ROLE_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, service.ErrCannotDeleteSystemRole) {
			utils.ErrorCode(w, http.StatusForbidden, "CANNOT_DELETE_SYSTEM_ROLE", err.Error())
			return
		}
		slog.Error("failed to delete role", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to delete role")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "role deleted successfully",
	})
}

// AssignPermissions handles PUT /api/v1/admin/roles/{id}/permissions
func (h *RoleHandler) AssignPermissions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.AssignPermissionsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	err := h.rbacSvc.AssignPermissionsToRole(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "ROLE_NOT_FOUND", err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated role with permissions
	role, err := h.rbacSvc.GetRoleByID(r.Context(), id)
	if err != nil {
		utils.JSON(w, http.StatusOK, map[string]string{
			"message": "role permissions updated successfully",
		})
		return
	}

	utils.JSON(w, http.StatusOK, role)
}

// ListPermissions handles GET /api/v1/admin/permissions
func (h *RoleHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.rbacSvc.ListPermissions(r.Context())
	if err != nil {
		slog.Error("failed to fetch permissions", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to fetch permissions")
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
			utils.ErrorCode(w, http.StatusNotFound, "PERMISSION_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to get permission detail", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to get permission detail")
		return
	}
	utils.JSON(w, http.StatusOK, perm)
}

// CreatePermission handles POST /api/v1/admin/permissions
func (h *RoleHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var input models.CreatePermissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	perm, err := h.rbacSvc.CreatePermission(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNameExists) {
			utils.ErrorCode(w, http.StatusConflict, "PERMISSION_NAME_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, perm)
}

// UpdatePermission handles PUT /api/v1/admin/permissions/{id}
func (h *RoleHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input models.UpdatePermissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	perm, err := h.rbacSvc.UpdatePermission(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrPermissionNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "PERMISSION_NOT_FOUND", err.Error())
			return
		}
		utils.ValidationError(w, err)
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
			utils.ErrorCode(w, http.StatusNotFound, "PERMISSION_NOT_FOUND", err.Error())
			return
		}
		slog.Error("failed to delete permission", slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to delete permission")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "permission deleted successfully",
	})
}
