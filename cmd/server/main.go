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
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
	"github.com/mochammadbagas/go-crud-api/internal/service"
)

func main() {
	// 1. Structured Logging Setup (JSON)
	initLogger()
	slog.Info("Memulai bootstrap mmdbconnect CMS (Blog, Article, Journal)...")

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

	// 5. Initialize Repositories (Database Access Layer via sqlc queries & database/sql)
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	postRepo := repository.NewPostRepository(db)

	// 6. Initialize Services (Business Logic Layer)
	markdownSvc := service.NewMarkdownService()
	authSvc := service.NewAuthService(userRepo, sessionRepo)
	categorySvc := service.NewCategoryService(categoryRepo)
	postSvc := service.NewPostService(postRepo, categoryRepo, markdownSvc)

	// 7. Initialize HTTP Handlers
	adminAuthHandler := handlers.NewAdminAuthHandler(authSvc, cfg)
	adminPostHandler := handlers.NewAdminPostHandler(postSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	adminUploadHandler := handlers.NewAdminUploadHandler(cfg)
	publicPostHandler := handlers.NewPublicPostHandler(postSvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, postSvc)

	// 8. Setup Chi HTTP Router
	router := routes.Setup(
		cfg,
		authSvc,
		adminAuthHandler,
		adminPostHandler,
		adminCategoryHandler,
		adminUploadHandler,
		publicPostHandler,
		publicCategoryHandler,
	)

	// 9. Start Background Periodic Session Cleanup
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go startSessionCleanup(cleanupCtx, sessionRepo, 1*time.Hour)

	// 10. Run HTTP Server with Graceful Shutdown
	runServer(router, cfg.Port)
}

func startSessionCleanup(ctx context.Context, sessionRepo repository.SessionRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial cleanup on startup
	if err := sessionRepo.DeleteExpired(ctx); err != nil {
		slog.Error("Gagal membersihkan sesi kadaluarsa saat start", slog.Any("error", err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sessionRepo.DeleteExpired(ctx); err != nil {
				slog.Error("Gagal membersihkan sesi kadaluarsa berkala", slog.Any("error", err))
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
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Server berhasil aktif dan siap melayani permintaan", slog.String("url", "http://localhost"+serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server mengalami kegagalan fatal", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Listen for OS termination signals (Ctrl+C, SIGTERM)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	<-shutdownSignal
	slog.Info("Sinyal shutdown diterima. Menghentikan server secara aman (graceful shutdown)...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Shutdown server gagal secara bersih", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Server berhasil dinonaktifkan dengan aman.")
}
