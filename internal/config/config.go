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
	AppEnv         string
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
	MailerDriver      string
	ResendAPIKey      string
	MailFrom          string
	FrontendURL       string
	StorageDriver     string
	S3Bucket          string
	S3Region          string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3PublicURL       string
}

// DefaultInsecureJWTSecret is the fallback secret used during local development.
const DefaultInsecureJWTSecret = "super-secret-starter-kit-jwt-key-change-in-production"

// Load loads configuration from environment variables or .env file.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info("Notice: .env file not found, falling back to system environment variables.")
	}

	appEnv := strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development")))
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

	jwtSecret := getEnv("JWT_SECRET", DefaultInsecureJWTSecret)
	if appEnv == "production" {
		if jwtSecret == DefaultInsecureJWTSecret {
			slog.Error("FATAL: Default insecure JWT_SECRET detected while APP_ENV is set to production. Please set a strong, unique JWT_SECRET!")
			os.Exit(1)
		}
		if len(jwtSecret) < 32 {
			slog.Error("FATAL: JWT_SECRET is too short (< 32 characters) for production environment. Minimum 32 characters required!")
			os.Exit(1)
		}
	}

	return &Config{
		AppEnv:         appEnv,
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
		JWTSecret:      jwtSecret,
		JWTAccessTTL:   time.Duration(accessTTLMin) * time.Minute,
		JWTRefreshTTL:  time.Duration(refreshTTLDays) * 24 * time.Hour,
		MailerDriver:      strings.ToLower(strings.TrimSpace(getEnv("MAILER_DRIVER", "log"))),
		ResendAPIKey:      getEnv("RESEND_API_KEY", ""),
		MailFrom:          getEnv("MAIL_FROM", "Go Starter Kit <no-reply@localhost>"),
		FrontendURL:       strings.TrimRight(getEnv("FRONTEND_URL", "http://localhost:3000"), "/"),
		StorageDriver:     strings.ToLower(strings.TrimSpace(getEnv("STORAGE_DRIVER", "local"))),
		S3Bucket:          getEnv("S3_BUCKET", ""),
		S3Region:          getEnv("S3_REGION", "auto"),
		S3Endpoint:        getEnv("S3_ENDPOINT", ""),
		S3AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3PublicURL:       strings.TrimRight(getEnv("S3_PUBLIC_URL", ""), "/"),
	}
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
