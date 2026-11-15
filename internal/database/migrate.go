package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mochammadbagas/go-crud-api/db"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations executes database migrations using golang-migrate.
func RunMigrations(sqlDB *sql.DB, dbName string, fresh bool) error {
	if fresh {
		slog.Warn("Flag -fresh aktif. Menjatuhkan (dropping) seluruh tabel database...")
		dropScript := `
			SET FOREIGN_KEY_CHECKS = 0;
			DROP TABLE IF EXISTS user_roles;
			DROP TABLE IF EXISTS role_permissions;
			DROP TABLE IF EXISTS permissions;
			DROP TABLE IF EXISTS roles;
			DROP TABLE IF EXISTS auth_tokens;
			DROP TABLE IF EXISTS refresh_tokens;
			DROP TABLE IF EXISTS user_profiles;
			DROP TABLE IF EXISTS sessions;
			DROP TABLE IF EXISTS post_views;
			DROP TABLE IF EXISTS posts;
			DROP TABLE IF EXISTS categories;
			DROP TABLE IF EXISTS users;
			DROP TABLE IF EXISTS schema_migrations;
			SET FOREIGN_KEY_CHECKS = 1;
		`
		if _, err := sqlDB.Exec(dropScript); err != nil {
			slog.Error("Gagal mereset tabel database", slog.Any("error", err))
		} else {
			slog.Info("Tabel lama berhasil dibersihkan.")
		}
	}

	sourceDriver, err := iofs.New(db.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("gagal menginisialisasi migration source: %w", err)
	}

	dbDriver, err := mysql.WithInstance(sqlDB, &mysql.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return fmt.Errorf("gagal menginisialisasi database driver migration: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver)
	if err != nil {
		return fmt.Errorf("gagal membuat migration instance: %w", err)
	}

	slog.Info("Menjalankan migrasi database via golang-migrate...")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("gagal mengeksekusi migrasi UP: %w", err)
	}

	slog.Info("Migrasi database berhasil dijalankan dan skema up-to-date.")
	return nil
}

