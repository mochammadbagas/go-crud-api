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

	"crudapi/internal/config"
	"crudapi/internal/database"
	"crudapi/internal/handlers"
	"crudapi/internal/routes"
	"crudapi/internal/store"
)

// main adalah titik masuk (Entry Point) utama aplikasi backend.
//
// Konsep Belajar Golang: Composition Root & High-Level Orchestration
//   - Fungsi `main()` yang ideal hanya bertindak sebagai "konduktor musik":
//     ia merakit dependensi tingkat tinggi dan tidak memuat logika detail SQL atau rute spesifik.
//   - Menjaga `main()` tetap ringkas membuat seluruh alur aplikasi terbaca seperti daftar isi.
func main() {
	// 1. Inisialisasi Structured Logger (JSON Format)
	initLogger()
	slog.Info("Memulai bootstrap aplikasi server...")

	// 2. Parsing CLI Flags
	fresh := flag.Bool("fresh", false, "Hapus seluruh tabel sebelum migrasi (seperti migrate:fresh)")
	flag.Parse()

	// 3. Pemuatan Konfigurasi Lingkungan (.env / OS Env)
	cfg := config.Load()

	// 4. Inisialisasi Database (Koneksi, Pool, Migrasi, & Seeder Otomatis)
	db, err := database.Init(cfg, *fresh)
	if err != nil {
		slog.Error("Inisialisasi database gagal", slog.Any("error", err))
		os.Exit(1)
	}

	// 5. Dependency Injection (Perakitan Repository & Handlers)
	productStore := store.NewMySQLStore(db)
	productHandler := handlers.NewProductHandler(productStore)
	authHandler := handlers.NewAuthHandler(productStore, cfg.JWTSecret)

	// 6. Inisialisasi Router Gin & Pemetaan Seluruh Rute API
	router := routes.Setup(productHandler, authHandler, cfg.JWTSecret)

	// 7. Menjalankan Server HTTP dengan Mekanisme Graceful Shutdown
	runServer(router, cfg.Port)
}

// initLogger mengonfigurasi logger terstruktur bawaan Go (`log/slog`) ke format JSON.
func initLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

// runServer mengonfigurasi HTTP Server dengan batas waktu (timeouts) yang aman
// dan mendengarkan sinyal terminasi OS (Ctrl+C / SIGTERM) untuk proses Graceful Shutdown.
//
// Konsep Belajar Golang:
// 1. Goroutine: Menjalankan `srv.ListenAndServe()` di thread ringan terpisah agar non-blocking.
// 2. Channel & OS Signal: Menahan proses utama sampai sinyal interupsi diterima dari sistem operasi.
// 3. Context Timeout: Memberikan masa tenggang 10 detik agar request yang sedang diproses selesai.
func runServer(handler http.Handler, port string) {
	serverAddr := ":" + port
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		WriteTimeout: 15 * time.Second, // Mencegah request write menggantung
		ReadTimeout:  15 * time.Second, // Mencegah serangan Slowloris
		IdleTimeout:  60 * time.Second,
	}

	// Jalankan server di goroutine terpisah
	go func() {
		slog.Info("Server berhasil aktif dan siap melayani request", slog.String("url", "http://localhost"+serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server mengalami kegagalan fatal", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Menunggu sinyal interrupt/terminasi dari OS (Ctrl+C, kill)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	<-shutdownSignal
	slog.Info("Sinyal shutdown diterima. Menghentikan server secara anggun (gracefully)...")

	// Timeout context 10 detik untuk proses shutdown bersih
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Shutdown server gagal secara bersih", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Server berhasil dinonaktifkan dengan aman.")
}
