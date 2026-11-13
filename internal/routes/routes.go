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
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// Setup sets up Chi router with all public and admin endpoints.
func Setup(
	cfg *config.Config,
	authSvc service.AuthServiceInterface,
	adminAuthHandler *handlers.AdminAuthHandler,
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

	// Static file server for local image uploads
	_ = os.MkdirAll(cfg.UploadDir, 0755)
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, cfg.UploadDir))
	fileServer(r, "/uploads", filesDir)

	// Section 7 URL Structure aliases
	r.Get("/blog/{slug}", publicPostHandler.Get)
	r.Get("/articles/{slug}", publicPostHandler.Get)
	r.Get("/journal/{slug}", publicPostHandler.Get)
	r.Get("/categories/{slug}", publicCategoryHandler.ListPostsByCategory)

	// API v1 Routes
	r.Route("/api/v1", func(v1 chi.Router) {
		// ---------------------------------------------------------------------
		// 1. PUBLIC ENDPOINTS (Section 6.1)
		// ---------------------------------------------------------------------
		v1.Get("/posts", publicPostHandler.List)
		v1.Get("/posts/{slug}", publicPostHandler.Get)
		v1.Get("/categories", publicCategoryHandler.List)
		v1.Get("/categories/{slug}/posts", publicCategoryHandler.ListPostsByCategory)

		// ---------------------------------------------------------------------
		// 2. ADMIN ENDPOINTS (Section 6.2)
		// ---------------------------------------------------------------------
		v1.Route("/admin", func(admin chi.Router) {
			// Public admin login with rate limiting (10 attempts/minute per IP)
			loginLimiter := httprate.LimitBy(
				10,
				time.Minute,
				clientIPKey,
				httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
					utils.Error(w, http.StatusTooManyRequests, "terlalu banyak percobaan login, silakan coba lagi nanti")
				}),
			)
			admin.With(loginLimiter).Post("/login", adminAuthHandler.Login)

			// Protected admin routes (Requires valid HTTP-only cookie session)
			admin.Group(func(protected chi.Router) {
				protected.Use(middleware.AuthMiddleware(authSvc))

				protected.Post("/logout", adminAuthHandler.Logout)
				protected.Get("/me", adminAuthHandler.Me)

				// Posts CRUD
				protected.Get("/posts", adminPostHandler.List)
				protected.Post("/posts", adminPostHandler.Create)
				protected.Get("/posts/{id}", adminPostHandler.Get)
				protected.Put("/posts/{id}", adminPostHandler.Update)
				protected.Delete("/posts/{id}", adminPostHandler.Delete)

				// Categories CRUD
				protected.Get("/categories", adminCategoryHandler.List)
				protected.Post("/categories", adminCategoryHandler.Create)
				protected.Put("/categories/{id}", adminCategoryHandler.Update)
				protected.Delete("/categories/{id}", adminCategoryHandler.Delete)

				// Image upload (Local Filesystem storage)
				protected.Post("/upload", adminUploadHandler.Upload)
			})
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
