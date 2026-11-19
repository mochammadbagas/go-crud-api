package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/database"
	"github.com/mochammadbagas/go-crud-api/internal/handlers"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/worker"
)

func main() {
	// 1. Structured Logging Setup (JSON)
	initLogger()
	slog.Info("Starting Go Starter Kit REST API bootstrap...")

	// 2. Command Line Flags
	fresh := flag.Bool("fresh", false, "Drop and recreate all database tables before running migrations")
	flag.Parse()

	// 3. Load Configurations
	cfg := config.Load()

	// 4. Initialize Database (Connection Pool, Migrations, & Seeder)
	db, err := database.Init(cfg, *fresh)
	if err != nil {
		slog.Error("Failed to initialize MySQL database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// 5. Initialize Repositories (ULID & database/sql via sqlc)
	userRepo := repository.NewUserRepository(db)
	profileRepo := repository.NewUserProfileRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authTokenRepo := repository.NewAuthTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	mediaRepo := repository.NewMediaRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// 6. Initialize Mailer (Pluggable: LogMailer for Dev / ResendMailer for Prod)
	mailSvc := mailer.New(cfg)

	// 7. Initialize Storage (Pluggable: LocalStorage for Dev / S3 or Cloudflare R2 for Prod)
	storageSvc := storage.New(cfg)

	// 8. Initialize Services (Business Logic Layer)
	authSvc := service.NewAuthService(userRepo, profileRepo, refreshTokenRepo, authTokenRepo, mailSvc, cfg, roleRepo)
	userSvc := service.NewUserService(userRepo, profileRepo, authTokenRepo, mailSvc, cfg)
	categorySvc := service.NewCategoryService(categoryRepo)
	tagSvc := service.NewTagService(tagRepo)
	mediaSvc := service.NewMediaService(mediaRepo, storageSvc, cfg)
	articleSvc := service.NewArticleService(articleRepo, categoryRepo, tagRepo)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo, articleRepo, cfg.JWTSecret)
	rbacSvc := service.NewRBACService(roleRepo, permRepo)
	adminUserSvc := service.NewAdminUserService(userRepo, profileRepo, roleRepo)

	// 9. Initialize HTTP Handlers
	authHandler := handlers.NewAuthHandler(authSvc, cfg)
	userHandler := handlers.NewUserHandler(userSvc, cfg, storageSvc)
	adminArticleHandler := handlers.NewAdminArticleHandler(articleSvc)
	publicArticleHandler := handlers.NewPublicArticleHandler(articleSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, articleSvc)
	tagHandler := handlers.NewTagHandler(tagSvc)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsSvc)
	roleHandler := handlers.NewRoleHandler(rbacSvc)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserSvc, storageSvc)

	// 10. Setup Chi HTTP Router
	router := routes.Setup(
		cfg,
		authHandler,
		userHandler,
		adminArticleHandler,
		publicArticleHandler,
		adminCategoryHandler,
		publicCategoryHandler,
		tagHandler,
		mediaHandler,
		analyticsHandler,
		roleHandler,
		adminUserHandler,
		userRepo,
		db,
	)

	// 11. Start Background Concurrency Workers with Graceful Shutdown
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()

	// Token cleanup (every 24 hours)
	go startTokenCleanup(cleanupCtx, refreshTokenRepo, authTokenRepo, 24*time.Hour)

	// Scheduled article publisher (every 1 minute)
	go worker.StartScheduledPublisher(cleanupCtx, articleRepo, 1*time.Minute)

	// Daily analytics aggregator (every 1 hour)
	go worker.StartAnalyticsAggregator(cleanupCtx, analyticsSvc, 1*time.Hour)

	// 12. Run HTTP Server with Graceful Shutdown
	runServer(router, cfg.Port)
}

func startTokenCleanup(ctx context.Context, tokenRepo repository.RefreshTokenRepository, authTokenRepo repository.AuthTokenRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial cleanup on startup
	if err := tokenRepo.DeleteExpired(ctx); err != nil {
		slog.Error("Failed to clean up expired refresh tokens on startup", slog.Any("error", err))
	}
	if err := authTokenRepo.DeleteExpiredOrUsed(ctx); err != nil {
		slog.Error("Failed to clean up expired auth tokens on startup", slog.Any("error", err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := tokenRepo.DeleteExpired(ctx); err != nil {
				slog.Error("Failed to clean up expired refresh tokens periodically", slog.Any("error", err))
			}
			if err := authTokenRepo.DeleteExpiredOrUsed(ctx); err != nil {
				slog.Error("Failed to clean up expired auth tokens periodically", slog.Any("error", err))
			}
		}
	}
}

func initLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func runServer(handler http.Handler, port string) {
	serverAddr := ":" + port
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("HTTP server is running and listening for requests", slog.String("port", port), slog.String("addr", serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server encountered a fatal error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("Shutdown signal received. Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Graceful server shutdown failed, forcing exit", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Server stopped safely.")
}
