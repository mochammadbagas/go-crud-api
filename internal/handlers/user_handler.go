package handlers

import (
	"encoding/json"
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
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusInternalServerError, "gagal memuat data profil")
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	updated, err := h.userSvc.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
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

	// 5 MB max memory
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		utils.Error(w, http.StatusBadRequest, "file avatar terlalu besar (maks 5MB) atau format tidak valid")
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
		utils.Error(w, http.StatusBadRequest, "file gambar avatar wajib disertakan (field 'avatar', 'image', atau 'file')")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		utils.Error(w, http.StatusBadRequest, "format avatar tidak didukung, gunakan JPG, PNG, atau WEBP")
		return
	}

	// Validate content via magic bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		utils.Error(w, http.StatusBadRequest, "gagal membaca header file gambar")
		return
	}

	mimeType := http.DetectContentType(buf[:n])
	allowedMIMEs := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowedMIMEs[mimeType] {
		utils.Error(w, http.StatusBadRequest, "tipe file tidak valid, hanya file gambar asli yang diizinkan")
		return
	}

	// Rewind file pointer
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			utils.Error(w, http.StatusInternalServerError, "gagal memproses stream file")
			return
		}
	} else {
		utils.Error(w, http.StatusInternalServerError, "stream file tidak mendukung seek")
		return
	}

	// Deteksi avatar lama jika ada sebelum mengunggah avatar baru
	var oldAvatarURL string
	if profile, err := h.userSvc.GetProfile(r.Context(), userID); err == nil && profile != nil && profile.AvatarURL != nil {
		oldAvatarURL = *profile.AvatarURL
	}

	// Rename nama file keseluruhan menggunakan ULID murni (tidak menyisakan nama file asal)
	uniqueName := fmt.Sprintf("avatar_%s%s", utils.NewULID(), ext)

	avatarURL, err := h.storage.Upload(r.Context(), file, uniqueName, mimeType)
	if err != nil {
		slog.Error("gagal menyimpan avatar pengguna", slog.String("filename", uniqueName), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menyimpan avatar")
		return
	}

	if err := h.userSvc.UpdateAvatar(r.Context(), userID, avatarURL); err != nil {
		// Rollback file yang baru diunggah jika database gagal update
		_ = h.storage.Delete(r.Context(), avatarURL)
		utils.Error(w, http.StatusInternalServerError, "gagal memperbarui avatar pengguna")
		return
	}

	// Auto-cleanup: Hapus avatar lama dari storage agar tidak menjadi orphaned file
	if oldAvatarURL != "" && oldAvatarURL != avatarURL {
		if delErr := h.storage.Delete(r.Context(), oldAvatarURL); delErr != nil {
			slog.Warn("Gagal menghapus avatar lama dari storage saat pembaruan", slog.String("old_avatar", oldAvatarURL), slog.Any("error", delErr))
		}
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message":    "avatar berhasil diunggah",
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

	// Ambil profil saat ini untuk mendapatkan URL avatar yang akan dihapus dari storage
	var currentAvatarURL string
	if profile, err := h.userSvc.GetProfile(r.Context(), userID); err == nil && profile != nil && profile.AvatarURL != nil {
		currentAvatarURL = *profile.AvatarURL
	}

	if err := h.userSvc.DeleteAvatar(r.Context(), userID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus avatar")
		return
	}

	// Auto-cleanup: Hapus file fisik avatar dari storage jika database berhasil direset
	if currentAvatarURL != "" {
		if delErr := h.storage.Delete(r.Context(), currentAvatarURL); delErr != nil {
			slog.Warn("Gagal menghapus avatar dari storage saat penghapusan", slog.String("avatar_url", currentAvatarURL), slog.Any("error", delErr))
		}
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "avatar berhasil dihapus",
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	if err := h.userSvc.UpdateUsername(r.Context(), userID, input); err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message":  "username berhasil diperbarui",
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
		utils.Error(w, http.StatusInternalServerError, "gagal memuat status akun")
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
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	if err := h.userSvc.RequestEmailChange(r.Context(), userID, input); err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "tautan konfirmasi telah dikirimkan ke alamat email baru Anda. Silakan periksa inbox.",
	})
}

// ConfirmEmailChange handles POST and GET /api/v1/users/email/confirm-change
func (h *UserHandler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		var input models.ConfirmEmailChangeInput
		_ = json.NewDecoder(r.Body).Decode(&input)
		token = input.Token
	}

	if strings.TrimSpace(token) == "" {
		utils.Error(w, http.StatusBadRequest, "token konfirmasi perubahan email wajib disertakan")
		return
	}

	if err := h.userSvc.ConfirmEmailChange(r.Context(), models.ConfirmEmailChangeInput{Token: token}); err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrAuthTokenInvalid) || errors.Is(err, service.ErrAuthTokenUsed) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "alamat email utama Anda berhasil diperbarui dan telah terverifikasi.",
	})
}
