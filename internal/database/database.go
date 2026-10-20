package database

import (
	"fmt"
	"log/slog"
	"time"

	"crudapi/internal/config"
	"crudapi/internal/models"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Init menginisialisasi seluruh siklus database:
// 1. Membuka koneksi MySQL via GORM
// 2. Mengonfigurasi Connection Pooling
// 3. Menjalankan Reset Tabel (jika flag -fresh aktif)
// 4. Menjalankan Auto-Migration untuk seluruh model
// 5. Menjalankan Seeder otomatis jika tabel masih kosong
//
// Konsep Belajar Golang:
// Mengenkapsulasi seluruh urusan database dalam satu fungsi inisialisasi membuat
// file `main.go` tidak terbebani oleh detail konfigurasi SQL tingkat rendah.
func Init(cfg *config.Config, fresh bool) (*gorm.DB, error) {
	// 1. Format DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	slog.Info("Membuka koneksi ke MySQL...", slog.String("host", cfg.DBHost), slog.String("port", cfg.DBPort))
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke database: %w", err)
	}

	// 2. Konfigurasi Connection Pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gagal mengakses generic database object: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)           // Maksimum koneksi idle siap pakai
	sqlDB.SetMaxOpenConns(100)          // Batas atas koneksi terbuka bersamaan
	sqlDB.SetConnMaxLifetime(time.Hour) // Umur maksimal sebuah koneksi sebelum di-refresh

	// 3. Reset database jika flag -fresh diaktifkan (drop all tables)
	if fresh {
		slog.Warn("Flag -fresh aktif. Menjatuhkan (dropping) seluruh tabel database...")
		db.Exec("SET FOREIGN_KEY_CHECKS = 0;")
		if err := db.Migrator().DropTable(&models.Product{}, &models.Category{}, &models.User{}); err != nil {
			slog.Error("Gagal menjatuhkan tabel", slog.Any("error", err))
		}
		db.Exec("SET FOREIGN_KEY_CHECKS = 1;")
		slog.Info("Tabel lama berhasil dibersihkan.")
	}

	// 4. Auto Migration skema tabel
	slog.Info("Menjalankan migrasi database otomatis...")
	if err := db.AutoMigrate(&models.Category{}, &models.Product{}, &models.User{}); err != nil {
		return nil, fmt.Errorf("gagal menjalankan auto-migrate: %w", err)
	}

	// 5. Eksekusi database seeder awal jika data masih kosong
	if err := Seed(db); err != nil {
		slog.Error("Gagal menjalankan database seeder", slog.Any("error", err))
	}

	return db, nil
}
