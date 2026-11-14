package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

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
	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration
	MailerDriver   string
	ResendAPIKey   string
	MailFrom       string
	FrontendURL    string
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

	accessTTLMin, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "15"))
	if accessTTLMin <= 0 {
		accessTTLMin = 15
	}

	refreshTTLDays, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL_DAYS", "7"))
	if refreshTTLDays <= 0 {
		refreshTTLDays = 7
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
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-starter-kit-jwt-key-change-in-production"),
		JWTAccessTTL:   time.Duration(accessTTLMin) * time.Minute,
		JWTRefreshTTL:  time.Duration(refreshTTLDays) * 24 * time.Hour,
		MailerDriver:   strings.ToLower(strings.TrimSpace(getEnv("MAILER_DRIVER", "log"))),
		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		MailFrom:       getEnv("MAIL_FROM", "Go Starter Kit <no-reply@localhost>"),
		FrontendURL:    strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
