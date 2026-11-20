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
		slog.Warn("Flag -fresh is active. Dropping all existing database tables...")
		tables := []string{
			"user_roles",
			"role_permissions",
			"permissions",
			"roles",
			"auth_tokens",
			"refresh_tokens",
			"user_profiles",
			"sessions",
			"article_daily_analytics",
			"article_analytics_events",
			"article_related",
			"article_revisions",
			"article_media",
			"article_tags",
			"articles",
			"media",
			"tags",
			"settings",
			"post_views",
			"posts",
			"categories",
			"users",
			"schema_migrations",
		}

		_, _ = sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 0")
		for _, table := range tables {
			if _, err := sqlDB.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				slog.Error("Failed to drop table", slog.String("table", table), slog.Any("error", err))
			}
		}
		_, _ = sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 1")
		slog.Info("Existing tables dropped successfully.")
	}

	sourceDriver, err := iofs.New(db.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to initialize migration source: %w", err)
	}

	dbDriver, err := mysql.WithInstance(sqlDB, &mysql.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database driver for migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	slog.Info("Running database migrations via golang-migrate...")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to execute UP migrations: %w", err)
	}

	slog.Info("Database migrations executed successfully, schema is up-to-date.")
	return nil
}

