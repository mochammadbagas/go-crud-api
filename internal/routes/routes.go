package routes

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/handlers"
	"github.com/mochammadbagas/go-crud-api/internal/middleware"
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
	adminArticleHandler *handlers.AdminArticleHandler,
	publicArticleHandler *handlers.PublicArticleHandler,
	adminCategoryHandler *handlers.AdminCategoryHandler,
	publicCategoryHandler *handlers.PublicCategoryHandler,
	tagHandler *handlers.TagHandler,
	mediaHandler *handlers.MediaHandler,
	analyticsHandler *handlers.AnalyticsHandler,
	roleHandler *handlers.RoleHandler,
	adminUserHandler *handlers.AdminUserHandler,
	userRepo repository.UserRepository,
	sqlDB ...*sql.DB,
) http.Handler {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger(cfg))
	r.Use(middleware.Recoverer())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Unified JSON 404 & 405 error handlers
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		utils.ErrorCode(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	// Comprehensive Health & Readiness Probes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if len(sqlDB) > 0 && sqlDB[0] != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := sqlDB[0].PingContext(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"degraded","database":"unreachable"}`))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","database":"healthy"}`))
	})
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"live"}`))
	})
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if len(sqlDB) > 0 && sqlDB[0] != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := sqlDB[0].PingContext(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"unavailable","database":"unreachable"}`))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready","database":"healthy"}`))
	})

	// Static file server for local image uploads
	_ = os.MkdirAll(cfg.UploadDir, 0755)
	uploadPath := cfg.UploadDir
	if !filepath.IsAbs(uploadPath) {
		workDir, _ := os.Getwd()
		uploadPath = filepath.Join(workDir, uploadPath)
	}
	filesDir := http.Dir(uploadPath)
	fileServer(r, "/uploads", filesDir)

	// SEO / URL Structure aliases
	r.Get("/blog/{slug}", publicArticleHandler.Get)
	r.Get("/articles/{slug}", publicArticleHandler.Get)
	r.Get("/journal/{slug}", publicArticleHandler.Get)
	r.Get("/categories/{slug}", publicCategoryHandler.ListArticlesByCategory)

	// API v1 Routes
	r.Route("/api/v1", func(v1 chi.Router) {
		// ---------------------------------------------------------------------
		// 1. PUBLIC CONTENT & ANALYTICS ENDPOINTS
		// ---------------------------------------------------------------------
		v1.Get("/articles", publicArticleHandler.List)
		v1.Get("/articles/{slug}", publicArticleHandler.Get)

		analyticsLimiter := httprate.LimitBy(
			60,
			time.Minute,
			clientIPKey,
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				utils.ErrorCode(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many analytics requests, please slow down")
			}),
		)
		v1.With(analyticsLimiter).Post("/articles/{id}/analytics", analyticsHandler.RecordEvent)

		v1.Get("/categories", publicCategoryHandler.List)
		v1.Get("/categories/{slug}/articles", publicCategoryHandler.ListArticlesByCategory)

		v1.Get("/tags", tagHandler.List)

		// Shared rate limiter for sensitive authentication & token verification endpoints
		authLimiter := httprate.LimitBy(
			10,
			time.Minute,
			clientIPKey,
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				utils.ErrorCode(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many authentication attempts, please try again later")
			}),
		)

		// ---------------------------------------------------------------------
		// 2. AUTHENTICATION ENDPOINTS (/api/v1/auth)
		// ---------------------------------------------------------------------
		v1.Route("/auth", func(auth chi.Router) {
			// Public auth routes (all rate-limited)
			auth.With(authLimiter).Post("/register", authHandler.Register)
			auth.With(authLimiter).Post("/login", authHandler.Login)
			auth.With(authLimiter).Post("/refresh", authHandler.RefreshToken)
			auth.With(authLimiter).Post("/logout", authHandler.Logout)
			auth.With(authLimiter).Post("/forgot-password", authHandler.ForgotPassword)
			auth.With(authLimiter).Post("/reset-password", authHandler.ResetPassword)
			auth.With(authLimiter).Post("/verify-email", authHandler.VerifyEmail)
			auth.With(authLimiter).Post("/resend-verification", authHandler.ResendVerification)

			// Protected auth routes (requires valid JWT Access Token)
			auth.Group(func(protected chi.Router) {
				protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

				protected.Post("/change-password", authHandler.ChangePassword)
			})
		})

		// ---------------------------------------------------------------------
		// 3. USER MANAGEMENT ENDPOINTS (/api/v1/users)
		// ---------------------------------------------------------------------
		v1.Route("/users", func(users chi.Router) {
			// Public email change confirmation (token-gated, POST only to prevent scanner consumption)
			users.With(authLimiter).Post("/email/confirm-change", userHandler.ConfirmEmailChange)

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

			// Articles CRUD
			admin.With(middleware.RequirePermission("articles.read", userRepo)).Get("/articles", adminArticleHandler.List)
			admin.With(middleware.RequirePermission("articles.create", userRepo)).Post("/articles", adminArticleHandler.Create)
			admin.With(middleware.RequirePermission("articles.read", userRepo)).Get("/articles/{id}", adminArticleHandler.Get)
			admin.With(middleware.RequirePermission("articles.update", userRepo)).Put("/articles/{id}", adminArticleHandler.Update)
			admin.With(middleware.RequirePermission("articles.delete", userRepo)).Delete("/articles/{id}", adminArticleHandler.Delete)
			admin.With(middleware.RequirePermission("articles.read", userRepo)).Get("/articles/{id}/revisions", adminArticleHandler.ListRevisions)
			admin.With(middleware.RequirePermission("analytics.read", userRepo)).Get("/articles/{id}/analytics", analyticsHandler.GetArticleAnalytics)

			// Categories CRUD
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Get("/categories", adminCategoryHandler.List)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Post("/categories", adminCategoryHandler.Create)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Get("/categories/{id}", adminCategoryHandler.Get)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Put("/categories/{id}", adminCategoryHandler.Update)
			admin.With(middleware.RequirePermission("categories.manage", userRepo)).Delete("/categories/{id}", adminCategoryHandler.Delete)

			// Tags CRUD
			admin.With(middleware.RequirePermission("tags.read", userRepo)).Get("/tags", tagHandler.List)
			admin.With(middleware.RequirePermission("tags.create", userRepo)).Post("/tags", tagHandler.Create)
			admin.With(middleware.RequirePermission("tags.read", userRepo)).Get("/tags/{id}", tagHandler.Get)
			admin.With(middleware.RequirePermission("tags.update", userRepo)).Put("/tags/{id}", tagHandler.Update)
			admin.With(middleware.RequirePermission("tags.delete", userRepo)).Delete("/tags/{id}", tagHandler.Delete)

			// Media Library Management (Replaces old /upload)
			admin.With(middleware.RequirePermission("media.read", userRepo)).Get("/media", mediaHandler.List)
			admin.With(middleware.RequirePermission("media.upload", userRepo)).Post("/media", mediaHandler.Upload)
			admin.With(middleware.RequirePermission("media.read", userRepo)).Get("/media/{id}", mediaHandler.Get)
			admin.With(middleware.RequirePermission("media.delete", userRepo)).Delete("/media/{id}", mediaHandler.Delete)

			// Analytics Dashboard Overview
			admin.With(middleware.RequirePermission("analytics.read", userRepo)).Get("/analytics/overview", analyticsHandler.GetOverview)

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

// noDirListingFS wraps an http.FileSystem to prevent directory browsing.
// If an index.html does not exist in the requested directory, Open returns os.ErrNotExist.
type noDirListingFS struct {
	fs http.FileSystem
}

func (nfs noDirListingFS) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if s.IsDir() {
		index := strings.TrimSuffix(path, "/") + "/index.html"
		fIdx, err := nfs.fs.Open(index)
		if err != nil {
			f.Close()
			return nil, os.ErrNotExist
		}
		_ = fIdx.Close()
	}
	return f, nil
}

// fileServer sets up a http.FileServer handler to serve static files from a http.FileSystem.
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("fileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(noDirListingFS{fs: root}))
		fs.ServeHTTP(w, r)
	})
}
