package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type contextKey string

const (
	ClaimsContextKey contextKey = "auth_claims"
	UserIDContextKey contextKey = "auth_user_id"
	RoleContextKey   contextKey = "auth_role"
)

// JWTAuthMiddleware verifies the JWT Bearer access token.
func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				utils.Error(w, http.StatusUnauthorized, "unauthorized: header Authorization 'Bearer <token>' wajib disertakan")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := utils.ValidateAccessToken(jwtSecret, tokenStr)
			if err != nil {
				utils.Error(w, http.StatusUnauthorized, "unauthorized: token tidak valid atau telah kedaluwarsa")
				return
			}

			// Inject claims into request context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleContextKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ensures that the authenticated user possesses the required role (with Super Admin wildcard bypass).
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil || (!claims.HasRole(role) && !claims.IsSuperAdmin()) {
				utils.Error(w, http.StatusForbidden, "forbidden: Anda tidak memiliki hak akses untuk resource ini")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoles ensures that the authenticated user possesses at least one of the specified roles (with Super Admin wildcard bypass).
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				utils.Error(w, http.StatusForbidden, "forbidden: Anda tidak memiliki hak akses untuk resource ini")
				return
			}
			if claims.IsSuperAdmin() {
				next.ServeHTTP(w, r)
				return
			}
			for _, requiredRole := range roles {
				if claims.HasRole(requiredRole) {
					next.ServeHTTP(w, r)
					return
				}
			}
			utils.Error(w, http.StatusForbidden, "forbidden: Anda tidak memiliki hak akses untuk resource ini")
		})
	}
}

// RequirePermission ensures that the authenticated user possesses the specific permission (with Super Admin wildcard bypass).
func RequirePermission(permission string, userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				utils.Error(w, http.StatusForbidden, "forbidden: Anda tidak memiliki hak akses untuk resource ini")
				return
			}

			// Super Admin wildcard bypass
			if claims.IsSuperAdmin() {
				next.ServeHTTP(w, r)
				return
			}

			perms, err := userRepo.GetUserPermissions(r.Context(), claims.UserID)
			if err != nil {
				utils.Error(w, http.StatusForbidden, "forbidden: gagal memverifikasi permission")
				return
			}

			hasPerm := false
			for _, p := range perms {
				if p == permission || p == "*" {
					hasPerm = true
					break
				}
			}

			if !hasPerm {
				utils.Error(w, http.StatusForbidden, "forbidden: Anda tidak memiliki hak akses untuk resource ini")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaimsFromContext retrieves JWT claims from request context.
func GetClaimsFromContext(ctx context.Context) *utils.JWTClaims {
	if val := ctx.Value(ClaimsContextKey); val != nil {
		if claims, ok := val.(*utils.JWTClaims); ok {
			return claims
		}
	}
	return nil
}

// GetUserIDFromContext retrieves the authenticated user's ID from context.
func GetUserIDFromContext(ctx context.Context) string {
	if val := ctx.Value(UserIDContextKey); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetRoleFromContext retrieves the authenticated user's role from context.
func GetRoleFromContext(ctx context.Context) string {
	if val := ctx.Value(RoleContextKey); val != nil {
		if role, ok := val.(string); ok {
			return role
		}
	}
	return ""
}
