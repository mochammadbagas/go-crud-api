package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

const RefreshTokenCookieName = "refresh_token"

type AuthHandler struct {
	authSvc service.AuthServiceInterface
	cfg     *config.Config
}

func NewAuthHandler(authSvc service.AuthServiceInterface, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authSvc: authSvc,
		cfg:     cfg,
	}
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input models.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.authSvc.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrEmailExists) || errors.Is(err, service.ErrUsernameExists) {
			utils.Error(w, http.StatusConflict, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, map[string]interface{}{
		"message": "registration successful",
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"status":     user.Status,
			"created_at": user.CreatedAt,
		},
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input models.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	userAgent := r.UserAgent()
	ipAddress := getClientIP(r)

	tokenResp, err := h.authSvc.Login(r.Context(), input, userAgent, ipAddress)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			utils.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotActive) {
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	// Set HTTP-only Cookie for web/SPA clients (Hybrid delivery)
	h.setRefreshTokenCookie(w, tokenResp.RefreshToken, int(h.cfg.JWTRefreshTTL.Seconds()))

	utils.JSON(w, http.StatusOK, tokenResp)
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	rawToken := h.extractRefreshToken(r)
	if rawToken == "" {
		utils.Error(w, http.StatusBadRequest, "refresh token not found in cookie or request body")
		return
	}

	userAgent := r.UserAgent()
	ipAddress := getClientIP(r)

	tokenResp, err := h.authSvc.RefreshToken(r.Context(), rawToken, userAgent, ipAddress)
	if err != nil {
		if errors.Is(err, service.ErrTokenReused) {
			h.clearRefreshTokenCookie(w)
			utils.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, service.ErrTokenExpired) || errors.Is(err, service.ErrUnauthorized) {
			h.clearRefreshTokenCookie(w)
			utils.Error(w, http.StatusUnauthorized, "refresh token is invalid or expired")
			return
		}
		if errors.Is(err, service.ErrUserNotActive) {
			h.clearRefreshTokenCookie(w)
			utils.Error(w, http.StatusForbidden, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update HTTP-only Cookie with rotated token
	h.setRefreshTokenCookie(w, tokenResp.RefreshToken, int(h.cfg.JWTRefreshTTL.Seconds()))

	utils.JSON(w, http.StatusOK, tokenResp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	rawToken := h.extractRefreshToken(r)
	if rawToken != "" {
		_ = h.authSvc.Logout(r.Context(), rawToken)
	}

	h.clearRefreshTokenCookie(w)

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input models.ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	if err := h.authSvc.ChangePassword(r.Context(), userID, input); err != nil {
		utils.ValidationError(w, err)
		return
	}

	// Clear refresh token cookie as sessions are revoked
	h.clearRefreshTokenCookie(w)

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "password changed successfully, all other sessions have been revoked. Please log in again.",
	})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input models.ForgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	if err := h.authSvc.ForgotPassword(r.Context(), input); err != nil {
		utils.ValidationError(w, err)
		return
	}

	// Always return anti-enumeration message
	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "If your email is registered in our system, password reset instructions have been sent.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input models.ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), input); err != nil {
		if errors.Is(err, service.ErrAuthTokenInvalid) || errors.Is(err, service.ErrAuthTokenUsed) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	h.clearRefreshTokenCookie(w)

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "password has been reset successfully. Please log in with your new password.",
	})
}

// VerifyEmail handles POST and GET /api/v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		var input models.VerifyEmailInput
		_ = json.NewDecoder(r.Body).Decode(&input)
		token = input.Token
	}

	if strings.TrimSpace(token) == "" {
		utils.Error(w, http.StatusBadRequest, "email verification token is required")
		return
	}

	if err := h.authSvc.VerifyEmail(r.Context(), models.VerifyEmailInput{Token: token}); err != nil {
		if errors.Is(err, service.ErrAuthTokenInvalid) || errors.Is(err, service.ErrAuthTokenUsed) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "email address verified successfully. Your account is now active.",
	})
}

// ResendVerification handles POST /api/v1/auth/resend-verification
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var input models.ResendVerificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	if err := h.authSvc.ResendVerification(r.Context(), input); err != nil {
		if errors.Is(err, service.ErrEmailAlreadyVerified) {
			utils.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{
		"message": "If your email is registered and unverified, a new verification link has been sent.",
	})
}

func (h *AuthHandler) extractRefreshToken(r *http.Request) string {
	// 1. Try reading from HTTP-only Cookie
	if cookie, err := r.Cookie(RefreshTokenCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}

	// 2. Try reading from X-Refresh-Token header
	if headerVal := r.Header.Get("X-Refresh-Token"); strings.TrimSpace(headerVal) != "" {
		return strings.TrimSpace(headerVal)
	}

	// 3. Try reading from JSON body
	var bodyInput models.RefreshTokenInput
	if err := json.NewDecoder(r.Body).Decode(&bodyInput); err == nil && strings.TrimSpace(bodyInput.RefreshToken) != "" {
		return strings.TrimSpace(bodyInput.RefreshToken)
	}

	return ""
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	return r.RemoteAddr
}
