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
)

func main() {
	// 1. Structured Logging Setup (JSON)
	initLogger()
	slog.Info("Memulai bootstrap Go Starter Kit REST API...")

	// 2. Command Line Flags
	fresh := flag.Bool("fresh", false, "Hapus seluruh tabel sebelum migrasi ulang")
	flag.Parse()

	// 3. Load Configurations
	cfg := config.Load()

	// 4. Initialize Database (Connection Pool, Migrations, & Seeder)
	db, err := database.Init(cfg, *fresh)
	if err != nil {
		slog.Error("Inisialisasi database MySQL gagal", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// 5. Initialize Repositories (ULID & database/sql via sqlc)
	userRepo := repository.NewUserRepository(db)
	profileRepo := repository.NewUserProfileRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authTokenRepo := repository.NewAuthTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	postRepo := repository.NewPostRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// 6. Initialize Mailer (Pluggable: LogMailer for Dev / ResendMailer for Prod)
	mailSvc := mailer.New(cfg)

	// 7. Initialize Services (Business Logic Layer)
	markdownSvc := service.NewMarkdownService()
	authSvc := service.NewAuthService(userRepo, profileRepo, refreshTokenRepo, authTokenRepo, mailSvc, cfg, roleRepo)
	userSvc := service.NewUserService(userRepo, profileRepo, authTokenRepo, mailSvc, cfg)
	categorySvc := service.NewCategoryService(categoryRepo)
	postSvc := service.NewPostService(postRepo, categoryRepo, markdownSvc)
	rbacSvc := service.NewRBACService(roleRepo, permRepo)
	adminUserSvc := service.NewAdminUserService(userRepo, profileRepo, roleRepo)

	// 8. Initialize HTTP Handlers
	authHandler := handlers.NewAuthHandler(authSvc, cfg)
	userHandler := handlers.NewUserHandler(userSvc, cfg)
	adminPostHandler := handlers.NewAdminPostHandler(postSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	adminUploadHandler := handlers.NewAdminUploadHandler(cfg)
	publicPostHandler := handlers.NewPublicPostHandler(postSvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, postSvc)
	roleHandler := handlers.NewRoleHandler(rbacSvc)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserSvc)

	// 9. Setup Chi HTTP Router
	router := routes.Setup(
		cfg,
		authHandler,
		userHandler,
		adminPostHandler,
		adminCategoryHandler,
		adminUploadHandler,
		publicPostHandler,
		publicCategoryHandler,
		roleHandler,
		adminUserHandler,
		userRepo,
	)

	// 10. Start Background Periodic Expired Tokens Cleanup (Every 24 hours)
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go startTokenCleanup(cleanupCtx, refreshTokenRepo, authTokenRepo, 24*time.Hour)

	// 11. Run HTTP Server with Graceful Shutdown
	runServer(router, cfg.Port)
}

func startTokenCleanup(ctx context.Context, tokenRepo repository.RefreshTokenRepository, authTokenRepo repository.AuthTokenRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial cleanup on startup
	if err := tokenRepo.DeleteExpired(ctx); err != nil {
		slog.Error("Gagal membersihkan refresh token kedaluwarsa saat start", slog.Any("error", err))
	}
	if err := authTokenRepo.DeleteExpiredOrUsed(ctx); err != nil {
		slog.Error("Gagal membersihkan auth token kedaluwarsa saat start", slog.Any("error", err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := tokenRepo.DeleteExpired(ctx); err != nil {
				slog.Error("Gagal membersihkan refresh token kedaluwarsa berkala", slog.Any("error", err))
			}
			if err := authTokenRepo.DeleteExpiredOrUsed(ctx); err != nil {
				slog.Error("Gagal membersihkan auth token kedaluwarsa berkala", slog.Any("error", err))
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
		slog.Info("Server HTTP aktif dan mendengarkan request", slog.String("port", port), slog.String("addr", serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server HTTP mengalami kegagalan fatal", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("Sinyal shutdown diterima. Menghentikan server secara graceful...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Shutdown server secara graceful gagal, memaksa berhenti", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Server berhasil dihentikan dengan aman.")
}
