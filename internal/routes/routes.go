package routes

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/handlers"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// Setup sets up Chi router with all public, auth, user, and admin endpoints.
func Setup(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	adminPostHandler *handlers.AdminPostHandler,
	adminCategoryHandler *handlers.AdminCategoryHandler,
	adminUploadHandler *handlers.AdminUploadHandler,
	publicPostHandler *handlers.PublicPostHandler,
	publicCategoryHandler *handlers.PublicCategoryHandler,
	roleHandler *handlers.RoleHandler,
	adminUserHandler *handlers.AdminUserHandler,
	userRepo repository.UserRepository,
) http.Handler {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Static file server for local image/avatar uploads
	_ = os.MkdirAll(cfg.UploadDir, 0755)
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, cfg.UploadDir))
	fileServer(r, "/uploads", filesDir)

	// SEO / URL Structure aliases
	r.Get("/blog/{slug}", publicPostHandler.Get)
	r.Get("/articles/{slug}", publicPostHandler.Get)
	r.Get("/journal/{slug}", publicPostHandler.Get)
	r.Get("/categories/{slug}", publicCategoryHandler.ListPostsByCategory)

	// API v1 Routes
	r.Route("/api/v1", func(v1 chi.Router) {
		// ---------------------------------------------------------------------
		// 1. PUBLIC CONTENT ENDPOINTS
		// ---------------------------------------------------------------------
		v1.Get("/posts", publicPostHandler.List)
		v1.Get("/posts/{slug}", publicPostHandler.Get)
		v1.Get("/categories", publicCategoryHandler.List)
		v1.Get("/categories/{slug}/posts", publicCategoryHandler.ListPostsByCategory)

		// ---------------------------------------------------------------------
		// 2. AUTHENTICATION ENDPOINTS (/api/v1/auth)
		// ---------------------------------------------------------------------
		v1.Route("/auth", func(auth chi.Router) {
			// Rate limiter for login & register (10 attempts/minute per IP)
			authLimiter := httprate.LimitBy(
				10,
				time.Minute,
				clientIPKey,
				httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
					utils.Error(w, http.StatusTooManyRequests, "terlalu banyak percobaan autentikasi, silakan coba lagi nanti")
				}),
			)

			// Public auth routes
			auth.With(authLimiter).Post("/register", authHandler.Register)
			auth.With(authLimiter).Post("/login", authHandler.Login)
			auth.Post("/refresh", authHandler.RefreshToken)
			auth.With(authLimiter).Post("/forgot-password", authHandler.ForgotPassword)
			auth.With(authLimiter).Post("/reset-password", authHandler.ResetPassword)
			auth.Post("/verify-email", authHandler.VerifyEmail)
			auth.Get("/verify-email", authHandler.VerifyEmail)
			auth.With(authLimiter).Post("/resend-verification", authHandler.ResendVerification)

			// Protected auth routes (requires valid JWT Access Token)
			auth.Group(func(protected chi.Router) {
				protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

				protected.Post("/logout", authHandler.Logout)
				protected.Post("/change-password", authHandler.ChangePassword)
			})
		})

		// ---------------------------------------------------------------------
		// 3. USER MANAGEMENT ENDPOINTS (/api/v1/users)
		// ---------------------------------------------------------------------
		v1.Route("/users", func(users chi.Router) {
			// Public email change confirmation (token-gated)
			users.Post("/email/confirm-change", userHandler.ConfirmEmailChange)
			users.Get("/email/confirm-change", userHandler.ConfirmEmailChange)

			// Protected user routes (requires valid JWT Access Token)
			users.Group(func(protected chi.Router) {
				protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

				protected.Get("/me", userHandler.GetProfile)
				protected.Put("/profile", userHandler.UpdateProfile)
				protected.Post("/avatar", userHandler.UploadAvatar)
				protected.Delete("/avatar", userHandler.DeleteAvatar)
				protected.Put("/username", userHandler.UpdateUsername)
				protected.Get("/status", userHandler.GetStatus)
				protected.Post("/email/request-change", userHandler.RequestEmailChange)
			})
		})

		// ---------------------------------------------------------------------
		// 4. ADMIN MANAGEMENT ENDPOINTS (/api/v1/admin)
		// ---------------------------------------------------------------------
		v1.Route("/admin", func(admin chi.Router) {
			admin.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
			admin.Use(middleware.RequireRoles(models.RoleAdmin, models.RoleSuperAdmin))

			// Posts CRUD
			admin.With(middleware.RequirePermission("posts.read", userRepo)).Get("/posts", adminPostHandler.List)
			admin.With(middleware.RequirePermission("posts.create", userRepo)).Post("/posts", adminPostHandler.Create)
			admin.With(middleware.RequirePermission("posts.read", userRepo)).Get("/posts/{id}", adminPostHandler.Get)
			admin.With(middleware.RequirePermission("posts.update", userRepo)).Put("/posts/{id}", adminPostHandler.Update)
			admin.With(middleware.RequirePermission("posts.delete", userRepo)).Delete("/posts/{id}", adminPostHandler.Delete)

			// Categories CRUD
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Get("/categories", adminCategoryHandler.List)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Post("/categories", adminCategoryHandler.Create)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Put("/categories/{id}", adminCategoryHandler.Update)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Delete("/categories/{id}", adminCategoryHandler.Delete)

			// General file/image upload
			admin.Post("/upload", adminUploadHandler.Upload)

			// Roles CRUD & Permissions assignment
			admin.With(middleware.RequirePermission("roles.read", userRepo)).Get("/roles", roleHandler.ListRoles)
			admin.With(middleware.RequirePermission("roles.create", userRepo)).Post("/roles", roleHandler.CreateRole)
			admin.With(middleware.RequirePermission("roles.read", userRepo)).Get("/roles/{id}", roleHandler.GetRole)
			admin.With(middleware.RequirePermission("roles.update", userRepo)).Put("/roles/{id}", roleHandler.UpdateRole)
			admin.With(middleware.RequirePermission("roles.delete", userRepo)).Delete("/roles/{id}", roleHandler.DeleteRole)
			admin.With(middleware.RequirePermission("roles.update", userRepo)).Put("/roles/{id}/permissions", roleHandler.AssignPermissions)

			// Permissions CRUD
			admin.With(middleware.RequirePermission("permissions.read", userRepo)).Get("/permissions", roleHandler.ListPermissions)
			admin.With(middleware.RequirePermission("permissions.create", userRepo)).Post("/permissions", roleHandler.CreatePermission)
			admin.With(middleware.RequirePermission("permissions.read", userRepo)).Get("/permissions/{id}", roleHandler.GetPermission)
			admin.With(middleware.RequirePermission("permissions.update", userRepo)).Put("/permissions/{id}", roleHandler.UpdatePermission)
			admin.With(middleware.RequirePermission("permissions.delete", userRepo)).Delete("/permissions/{id}", roleHandler.DeletePermission)

			// Admin Users Management
			admin.With(middleware.RequirePermission("users.read", userRepo)).Get("/users", adminUserHandler.ListUsers)
			admin.With(middleware.RequirePermission("users.read", userRepo)).Get("/users/{id}", adminUserHandler.GetUser)
			admin.With(middleware.RequirePermission("users.update", userRepo)).Put("/users/{id}", adminUserHandler.UpdateUser)
			admin.With(middleware.RequirePermission("users.update", userRepo)).Put("/users/{id}/status", adminUserHandler.UpdateStatus)
			admin.With(middleware.RequirePermission("users.update", userRepo)).Put("/users/{id}/roles", adminUserHandler.AssignRoles)
			admin.With(middleware.RequirePermission("users.delete", userRepo)).Delete("/users/{id}", adminUserHandler.DeleteUser)
		})
	})

	return r
}

func clientIPKey(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return httprate.CanonicalizeIP(ip), nil
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
