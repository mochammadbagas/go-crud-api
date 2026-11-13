package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type contextKey string

const (
	UserContextKey    contextKey = "auth_user"
	SessionContextKey contextKey = "auth_session"
	CookieSessionName string     = "session_id"
)

// SessionValidator defines the contract required to validate sessions in middleware.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID string) (*models.Session, error)
}

// AuthMiddleware verifies the secure HTTP-only cookie session.
func AuthMiddleware(authSvc SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var sessionID string

			// 1. Try reading from HTTP-only cookie
			if cookie, err := r.Cookie(CookieSessionName); err == nil && cookie.Value != "" {
				sessionID = cookie.Value
			}

			// 2. Fallback to Authorization: Bearer <session_id> for REST/HTTP clients like Postman/api.http
			if sessionID == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					sessionID = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if sessionID == "" {
				utils.Error(w, http.StatusUnauthorized, "unauthorized: sesi tidak ditemukan")
				return
			}

			// 3. Validate session in database
			session, err := authSvc.ValidateSession(r.Context(), sessionID)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "unauthorized: sesi tidak valid atau telah kadaluarsa")
				return
			}

			// 4. Inject session and user into context
			ctx := context.WithValue(r.Context(), SessionContextKey, session)
			ctx = context.WithValue(ctx, UserContextKey, session.User)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves authenticated user from request context.
func GetUserFromContext(ctx context.Context) *models.User {
	if val := ctx.Value(UserContextKey); val != nil {
		if user, ok := val.(*models.User); ok {
			return user
		}
	}
	return nil
}

// GetSessionFromContext retrieves active session from request context.
func GetSessionFromContext(ctx context.Context) *models.Session {
	if val := ctx.Value(SessionContextKey); val != nil {
		if sess, ok := val.(*models.Session); ok {
			return sess
		}
	}
	return nil
}
