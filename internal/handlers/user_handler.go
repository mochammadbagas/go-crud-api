package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type UserHandler struct {
	userSvc service.UserServiceInterface
	cfg     *config.Config
	storage storage.Storage
}

func NewUserHandler(userSvc service.UserServiceInterface, cfg *config.Config, store storage.Storage) *UserHandler {
	return &UserHandler{
		userSvc: userSvc,
		cfg:     cfg,
		storage: store,
	}
}

// GetProfile handles GET /api/v1/users/me
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.userSvc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
			return
		}
		utils.Error(w, http.StatusInternalServerError, "failed to load profile data")
		return
	}

	utils.JSON(w, http.StatusOK, profile)
}

// UpdateProfile handles PUT /api/v1/users/profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input models.UpdateProfileInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	updated, err := h.userSvc.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, updated)
}

// UploadAvatar handles POST /api/v1/users/avatar
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Enforce 5 MB maximum request stream size to prevent disk/memory exhaustion DoS
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "avatar file is too large (max 5MB) or invalid format")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		file, header, err = r.FormFile("image")
	}
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "avatar image file is required (field 'avatar', 'image', or 'file')")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		utils.Error(w, http.StatusBadRequest, "unsupported avatar format, please use JPG, PNG, or WEBP")
		return
	}

	// Validate content via magic bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		utils.Error(w, http.StatusBadRequest, "failed to read image file header")
		return
	}

	mimeType := http.DetectContentType(buf[:n])
	allowedMIMEs := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowedMIMEs[mimeType] {
		utils.Error(w, http.StatusBadRequest, "invalid file type, only authentic image files are allowed")
		return
	}

	// Rewind file pointer
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			utils.Error(w, http.StatusInternalServerError, "failed to process file stream")
			return
		}
	} else {
		utils.Error(w, http.StatusInternalServerError, "file stream does not support seek")
		return
	}

	// Detect old avatar if any before uploading new one
	var oldAvatarURL string
	if profile, err := h.userSvc.GetProfile(r.Context(), userID); err == nil && profile != nil && profile.AvatarURL != nil {
		oldAvatarURL = *profile.AvatarURL
	}

	// Rename file entirely using pure ULID (no original filename preserved)
	uniqueName := fmt.Sprintf("avatar_%s%s", utils.NewULID(), ext)

	avatarURL, err := h.storage.Upload(r.Context(), file, uniqueName, mimeType)
	if err != nil {
		slog.Error("failed to save user avatar", slog.String("filename", uniqueName), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}

	if err := h.userSvc.UpdateAvatar(r.Context(), userID, avatarURL); err != nil {
		// Rollback newly uploaded file if database update fails
		_ = h.storage.Delete(r.Context(), avatarURL)
		utils.Error(w, http.StatusInternalServerError, "failed to update user avatar")
		return
	}

	// Auto-cleanup: Delete old avatar from storage to avoid orphaned files
	if oldAvatarURL != "" && oldAvatarURL != avatarURL {
		if delErr := h.storage.Delete(r.Context(), oldAvatarURL); delErr != nil {
			slog.Warn("Failed to delete old avatar from storage during update", slog.String("old_avatar", oldAvatarURL), slog.Any("error", delErr))
		}
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message":    "avatar uploaded successfully",
		"avatar_url": avatarURL,
	})
}

// DeleteAvatar handles DELETE /api/v1/users/avatar
func (h *UserHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Fetch current profile to obtain avatar URL before deletion
	var currentAvatarURL string
	if profile, err := h.userSvc.GetProfile(r.Context(), userID); err == nil && profile != nil && profile.AvatarURL != nil {
		currentAvatarURL = *profile.AvatarURL
	}

	if err := h.userSvc.DeleteAvatar(r.Context(), userID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to delete avatar")
		return
	}

	// Auto-cleanup: Delete physical avatar file from storage once database record is cleared
	if currentAvatarURL != "" {
		if delErr := h.storage.Delete(r.Context(), currentAvatarURL); delErr != nil {
			slog.Warn("Failed to delete avatar from storage during removal", slog.String("avatar_url", currentAvatarURL), slog.Any("error", delErr))
		}
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "avatar deleted successfully",
	})
}

// UpdateUsername handles PUT /api/v1/users/username
func (h *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input models.UpdateUsernameInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	if err := h.userSvc.UpdateUsername(r.Context(), userID, input); err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			utils.ErrorCode(w, http.StatusConflict, "USERNAME_ALREADY_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message":  "username updated successfully",
		"username": input.Username,
	})
}

// GetStatus handles GET /api/v1/users/status
func (h *UserHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.userSvc.GetProfile(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to load account status")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"user_id":           profile.ID,
		"username":          profile.Username,
		"role":              profile.Role,
		"status":            profile.Status,
		"email_verified_at": profile.EmailVerifiedAt,
		"is_email_verified": profile.EmailVerifiedAt != nil,
	})
}

// RequestEmailChange handles POST /api/v1/users/email/request-change
func (h *UserHandler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input models.RequestEmailChangeInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	if err := h.userSvc.RequestEmailChange(r.Context(), userID, input); err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			utils.ErrorCode(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "a confirmation link has been sent to your new email address. Please check your inbox.",
	})
}

// ConfirmEmailChange handles POST and GET /api/v1/users/email/confirm-change
func (h *UserHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		var input models.ConfirmEmailChangeInput
		_ = utils.DecodeJSON(w, r, &input)
		token = input.Token
	}

	if strings.TrimSpace(token) == "" {
		utils.ErrorCode(w, http.StatusBadRequest, "TOKEN_REQUIRED", "email change confirmation token is required")
		return
	}

	if err := h.userSvc.ConfirmEmailChange(r.Context(), models.ConfirmEmailChangeInput{Token: token}); err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			utils.ErrorCode(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error())
			return
		}
		if errors.Is(err, service.ErrAuthTokenInvalid) {
			utils.ErrorCode(w, http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN", err.Error())
			return
		}
		if errors.Is(err, service.ErrAuthTokenUsed) {
			utils.ErrorCode(w, http.StatusBadRequest, "TOKEN_ALREADY_USED", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "your primary email address has been updated and verified successfully.",
	})
}
