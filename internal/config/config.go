package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config represents all application configuration loaded at runtime.
type Config struct {
	Port           string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	CookieSecure   bool
	CookieDomain   string
	UploadDir      string
	AdminName      string
	AdminEmail     string
	AdminPassword  string
	AllowedOrigins []string
}

// Load loads configuration from environment variables or .env file.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info("Catatan: File .env tidak ditemukan, menggunakan nilai environment bawaan/sistem.")
	}

	cookieSecure, _ := strconv.ParseBool(getEnv("COOKIE_SECURE", "false"))

	originsRaw := getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173")
	var allowedOrigins []string
	for _, origin := range strings.Split(originsRaw, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "go_crud_api"),
		CookieSecure:   cookieSecure,
		CookieDomain:   getEnv("COOKIE_DOMAIN", ""),
		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		AdminName:      getEnv("ADMIN_NAME", "Administrator"),
		AdminEmail:     getEnv("ADMIN_EMAIL", "admin@mmdbconnect.com"),
		AdminPassword:  getEnv("ADMIN_PASSWORD", "adminpassword123"),
		AllowedOrigins: allowedOrigins,
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
