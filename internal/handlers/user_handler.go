package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type UserHandler struct {
	userSvc service.UserServiceInterface
	cfg     *config.Config
}

func NewUserHandler(userSvc service.UserServiceInterface, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userSvc: userSvc,
		cfg:     cfg,
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
	}

	if err := os.MkdirAll(h.cfg.UploadDir, 0755); err != nil {
		utils.Error(w, http.StatusInternalServerError, "gagal menyiapkan direktori penyimpanan")
		return
	}

	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	uniqueName := fmt.Sprintf("avatar_%s_%x%s", userID, randomBytes, ext)
	dstPath := filepath.Join(h.cfg.UploadDir, uniqueName)

	dst, err := os.Create(dstPath)
	if err != nil {
		slog.Error("gagal membuat file avatar", slog.String("path", dstPath), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menyimpan avatar ke disk")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("gagal menyalin konten avatar", slog.String("path", dstPath), slog.Any("error", err))
		utils.Error(w, http.StatusInternalServerError, "gagal menyalin avatar")
		return
	}

	avatarURL := "/uploads/" + uniqueName
	if err := h.userSvc.UpdateAvatar(r.Context(), userID, avatarURL); err != nil {
		utils.Error(w, http.StatusInternalServerError, "gagal memperbarui avatar pengguna")
		return
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

	if err := h.userSvc.DeleteAvatar(r.Context(), userID); err != nil {
		utils.Error(w, http.StatusInternalServerError, "gagal menghapus avatar")
		return
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
