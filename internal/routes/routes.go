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
			users.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

			users.Get("/me", userHandler.GetProfile)
			users.Put("/profile", userHandler.UpdateProfile)
			users.Post("/avatar", userHandler.UploadAvatar)
			users.Delete("/avatar", userHandler.DeleteAvatar)
			users.Put("/username", userHandler.UpdateUsername)
			users.Get("/status", userHandler.GetStatus)
		})

		// ---------------------------------------------------------------------
		// 4. ADMIN MANAGEMENT ENDPOINTS (/api/v1/admin)
		// ---------------------------------------------------------------------
		v1.Route("/admin", func(admin chi.Router) {
			admin.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
			admin.Use(middleware.RequireRole(models.RoleAdmin))

			// Posts CRUD
			admin.Get("/posts", adminPostHandler.List)
			admin.Post("/posts", adminPostHandler.Create)
			admin.Get("/posts/{id}", adminPostHandler.Get)
			admin.Put("/posts/{id}", adminPostHandler.Update)
			admin.Delete("/posts/{id}", adminPostHandler.Delete)

			// Categories CRUD
			admin.Get("/categories", adminCategoryHandler.List)
			admin.Post("/categories", adminCategoryHandler.Create)
			admin.Put("/categories/{id}", adminCategoryHandler.Update)
			admin.Delete("/categories/{id}", adminCategoryHandler.Delete)

			// General file/image upload
			admin.Post("/upload", adminUploadHandler.Upload)
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
