package db

import "embed"

// MigrationsFS embeds all database migration SQL files.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS

