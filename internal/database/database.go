package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

// Init initializes the MySQL database connection, pool, migrations, and seed data.
func Init(cfg *config.Config, fresh bool) (*sql.DB, error) {
	// DSN format with UTC timezone, parseTime=true, and utf8mb4 charset
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_0900_ai_ci&parseTime=true&loc=UTC&multiStatements=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	slog.Info("Connecting to MySQL database...", slog.String("host", cfg.DBHost), slog.String("port", cfg.DBPort), slog.String("database", cfg.DBName))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Connection Pool configuration
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL server: %w", err)
	}

	// Run migrations
	if err := RunMigrations(db, cfg.DBName, fresh); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Seed initial data if empty
	if err := Seed(db, cfg); err != nil {
		return nil, fmt.Errorf("failed to run database seeder: %w", err)
	}

	return db, nil
}
