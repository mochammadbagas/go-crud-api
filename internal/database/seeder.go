package database

import (
	"log/slog"

	"crudapi/internal/models"

	"gorm.io/gorm"
)

// Seed bertugas menyemai (seeding) data awal ke database jika data masih kosong.
//
// Konsep Belajar Golang & Clean Architecture:
//  1. Single Responsibility Principle (SRP):
//     Memisahkan data dummy dan logika seeding dari file `main.go` membuat kode server
//     tetap bersih, mudah dibaca, dan mudah diperluas jika suatu saat jumlah data seeder bertambah banyak.
//  2. Idempotent Seeding:
//     Seeder memeriksa terlebih dahulu apakah data sudah ada (`count == 0`).
//     Jika tabel sudah berisi data, seeder tidak akan melakukan duplikasi data.
func Seed(db *gorm.DB) error {
	var categoryCount int64
	if err := db.Model(&models.Category{}).Count(&categoryCount).Error; err != nil {
		return err
	}

	// Jika database belum memiliki kategori sama sekali, lakukan pengisian data awal
	if categoryCount == 0 {
		slog.Info("Database kosong, menjalankan seeder data awal kategori dan produk...")

		// 1. Data Kategori
		elektronik := models.Category{Name: "Elektronik"}
		aksesoris := models.Category{Name: "Aksesoris"}

		if err := db.Create(&elektronik).Error; err != nil {
			return err
		}
		if err := db.Create(&aksesoris).Error; err != nil {
			return err
		}

		// 2. Data Produk (berelasi dengan Kategori yang baru dibuat)
		products := []models.Product{
			{
				Name:       "Keyboard Mekanikal RGB",
				Price:      450000,
				Stock:      12,
				CategoryID: aksesoris.ID,
			},
			{
				Name:       "Mouse Wireless Ergonomis",
				Price:      150000,
				Stock:      30,
				CategoryID: aksesoris.ID,
			},
			{
				Name:       "Televisi Smart LED 32 Inch",
				Price:      2200000,
				Stock:      5,
				CategoryID: elektronik.ID,
			},
		}

		if err := db.Create(&products).Error; err != nil {
			return err
		}

		slog.Info("Seeder database berhasil dijalankan.")
	}

	return nil
}
