package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type AdminAuthHandler struct {
	authSvc service.AuthServiceInterface
	cfg     *config.Config
}

func NewAdminAuthHandler(authSvc service.AuthServiceInterface, cfg *config.Config) *AdminAuthHandler {
	return &AdminAuthHandler{
		authSvc: authSvc,
		cfg:     cfg,
	}
}

// Login handles POST /api/v1/admin/login
func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input models.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "format body request tidak valid")
		return
	}

	session, err := h.authSvc.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			utils.Error(w, http.StatusUnauthorized, "email atau password salah")
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Set Secure HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieSessionName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600, // 7 days
	})

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"message":    "login berhasil",
		"session_id": session.ID,
		"user": models.UserProfile{
			ID:        session.User.ID,
			Name:      session.User.Name,
			Email:     session.User.Email,
			CreatedAt: session.User.CreatedAt,
		},
	})
}

// Logout handles POST /api/v1/admin/logout
func (h *AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var sessionID string
	if cookie, err := r.Cookie(middleware.CookieSessionName); err == nil {
		sessionID = cookie.Value
	}
	if sessionID == "" {
		if sess := middleware.GetSessionFromContext(r.Context()); sess != nil {
			sessionID = sess.ID
		}
	}

	_ = h.authSvc.Logout(r.Context(), sessionID)

	// Clear Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieSessionName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "logout berhasil",
	})
}

// Me handles GET /api/v1/admin/me
func (h *AdminAuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	utils.JSON(w, http.StatusOK, models.UserProfile{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

