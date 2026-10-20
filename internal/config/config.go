package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// Config merepresentasikan seluruh konfigurasi aplikasi yang dibutuhkan saat runtime.
// Mengumpulkan semua variabel environment dalam satu struct mempermudah
// pengelolaan konfigurasi terpusat (mengikuti metodologi 12-Factor App).
type Config struct {
	Port       string // Port tempat server HTTP berjalan (misal: "8080")
	DBHost     string // Hostname atau IP database MySQL (misal: "127.0.0.1")
	DBPort     string // Port database MySQL (default: "3306")
	DBUser     string // Username kredensial database (default: "root")
	DBPassword string // Password kredensial database
	DBName     string // Nama database yang digunakan (misal: "go_crud_api")
	JWTSecret  string // Kunci rahasia untuk menandatangani (signing) token JWT
}

// Load bertugas memuat variabel konfigurasi dari file .env (jika ada)
// atau mengambil nilai langsung dari environment variables sistem operasi.
//
// Konsep Belajar Golang:
//  1. Pada tahap development lokal, kita biasanya memakai file `.env`.
//  2. Pada tahap production (Docker, Kubernetes, AWS), variabel environment disuntikkan
//     langsung oleh environment OS/container. Oleh karena itu, jika file `.env` tidak ditemukan,
//     aplikasi tidak boleh crash/panic, melainkan cukup mencatat info dan beralih ke environment OS.
func Load() *Config {
	// Memuat file .env ke environment sistem operasi
	if err := godotenv.Load(); err != nil {
		slog.Info("Catatan: File .env tidak ditemukan, menggunakan nilai environment bawaan/sistem.")
	}

	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "go_crud_api"),
		JWTSecret:  getEnv("JWT_SECRET", "super_secret_key_12345"),
	}
}

// getEnv adalah fungsi pembantu (helper) untuk mengambil nilai environment variable.
//
// Konsep Belajar Golang:
//   - `os.LookupEnv` mengembalikan dua nilai: (value, exists).
//   - Ini adalah idiom Go untuk membedakan antara variabel yang "memang tidak diset"
//     dengan variabel yang "diset tapi bernilai string kosong".
//   - Jika key tidak ditemukan, kita kembalikan nilai cadangan (fallback).
func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
