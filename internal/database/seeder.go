package database

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

// Seed populates the database with initial admin, roles, permissions, categories, and sample posts.
func Seed(db *sql.DB, cfg *config.Config) error {
	now := time.Now().UTC()

	// 1. Seed Roles and Permissions (RBAC)
	roleMap, err := seedRBAC(db, now)
	if err != nil {
		return err
	}

	// 2. Seed Administrator if none exists
	var userCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return err
	}

	var adminID string
	if userCount == 0 {
		slog.Info("Seeding initial administrator account...")
		passwordHash, err := argon2id.CreateHash(cfg.AdminPassword, argon2id.DefaultParams)
		if err != nil {
			return err
		}

		adminID = utils.NewULID()
		_, err = db.Exec(
			`INSERT INTO users (id, username, email, password_hash, role, status, email_verified_at, created_at, updated_at) 
			 VALUES (?, ?, ?, ?, 'super_admin', 'active', ?, ?, ?)`,
			adminID, "admin", cfg.AdminEmail, passwordHash, now, now, now,
		)
		if err != nil {
			return err
		}

		_, err = db.Exec(
			`INSERT INTO user_profiles (user_id, full_name, bio, created_at, updated_at) 
			 VALUES (?, ?, ?, ?, ?)`,
			adminID, cfg.AdminName, "System Administrator", now, now,
		)
		if err != nil {
			return err
		}

		slog.Info("Administrator account created successfully", slog.String("email", cfg.AdminEmail), slog.String("id", adminID))
	} else {
		// Find existing admin ID by email
		_ = db.QueryRow("SELECT id FROM users WHERE email = ? LIMIT 1", cfg.AdminEmail).Scan(&adminID)
	}

	// Ensure admin user has super_admin role in user_roles
	if adminID != "" {
		if superAdminRoleID, ok := roleMap["super_admin"]; ok {
			_, _ = db.Exec(
				"INSERT IGNORE INTO user_roles (user_id, role_id, created_at) VALUES (?, ?, ?)",
				adminID, superAdminRoleID, now,
			)
			// Also update legacy role column to super_admin
			_, _ = db.Exec("UPDATE users SET role = 'super_admin' WHERE id = ?", adminID)
		}
	}

	var categoryCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount); err != nil {
		return err
	}

	// 3. Seed Categories & Posts if empty
	if categoryCount == 0 {
		slog.Info("Seeding initial categories and sample posts with ULID...")

		cat1ID := utils.NewULID()
		_, err := db.Exec("INSERT INTO categories (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			cat1ID, "Software Engineering", "software-engineering", now, now)
		if err != nil {
			return err
		}

		cat2ID := utils.NewULID()
		_, err = db.Exec("INSERT INTO categories (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			cat2ID, "Tutorials & Tech", "tutorials-tech", now, now)
		if err != nil {
			return err
		}

		cat3ID := utils.NewULID()
		_, err = db.Exec("INSERT INTO categories (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			cat3ID, "Personal Journey", "personal-journey", now, now)
		if err != nil {
			return err
		}

		// Sample Posts
		tags1, _ := json.Marshal([]string{"Go", "MySQL", "REST API", "Backend"})
		tags2, _ := json.Marshal([]string{"Career", "Reflection", "Growth"})
		tags3, _ := json.Marshal([]string{"Go", "Rust", "Benchmarking", "Performance"})

		// Article
		_, err = db.Exec(`
			INSERT INTO posts (
				id, type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			utils.NewULID(),
			"article",
			"Building REST APIs with Go",
			"building-rest-apis-with-go",
			"Comprehensive guide to designing and building high-performance RESTful APIs using Go and Chi.",
			"# Building REST APIs with Go\n\nGo is a highly efficient language for building modern web services...\n\n```go\npackage main\n\nfunc main() {\n    println(\"Hello, CMS!\")\n}\n```",
			nil,
			cat2ID,
			tags1,
			"Building REST APIs with Go - Complete Guide",
			"In-depth tutorial on building REST APIs with Go, MySQL, and clean modular architecture.",
			true,
			"published",
			now,
			now,
			now,
		)
		if err != nil {
			return err
		}

		// Blog
		_, err = db.Exec(`
			INSERT INTO posts (
				id, type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			utils.NewULID(),
			"blog",
			"My Development Journey",
			"my-development-journey",
			"Reflections on the journey of building backend systems and products over the years.",
			"# My Development Journey\n\nBeing a software engineer is a continuous learning journey...",
			nil,
			cat3ID,
			tags2,
			nil,
			nil,
			false,
			"published",
			now,
			now,
			now,
		)
		if err != nil {
			return err
		}

		// Journal
		_, err = db.Exec(`
			INSERT INTO posts (
				id, type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			utils.NewULID(),
			"journal",
			"Rust vs Go File Hashing",
			"rust-vs-go-file-hashing",
			"Benchmark notes comparing large file hashing performance between Rust and Go.",
			"# Rust vs Go File Hashing\n\nToday's experiment tests I/O throughput and CPU utilization...",
			nil,
			cat1ID,
			tags3,
			nil,
			nil,
			false,
			"published",
			now,
			now,
			now,
		)
		if err != nil {
			return err
		}

		// Draft Post
		_, err = db.Exec(`
			INSERT INTO posts (
				id, type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			utils.NewULID(),
			"article",
			"Upcoming: Event Driven Architecture in Go",
			"upcoming-event-driven-architecture-in-go",
			"Draft article exploring message queues and event streaming patterns.",
			"# Event Driven Architecture\n\nThis content is currently in draft status...",
			nil,
			cat1ID,
			tags1,
			nil,
			nil,
			false,
			"draft",
			nil,
			now,
			now,
		)
		if err != nil {
			return err
		}

		slog.Info("Initial categories and sample posts seeded successfully.")
	}

	return nil
}

type roleSeed struct {
	Name        string
	DisplayName string
	Description string
}

type permissionSeed struct {
	Name        string
	Description string
}

func seedRBAC(db *sql.DB, now time.Time) (map[string]string, error) {
	slog.Info("Seeding initial RBAC master data (Roles & Permissions)...")

	// 1. Roles
	roles := []roleSeed{
		{Name: "super_admin", DisplayName: "Super Admin", Description: "Full administrative access to all system resources"},
		{Name: "admin", DisplayName: "Admin", Description: "Manages users, content, and categories based on granted permissions"},
		{Name: "user", DisplayName: "User", Description: "Standard application user"},
	}

	roleMap := make(map[string]string) // name -> ID
	for _, r := range roles {
		var id string
		err := db.QueryRow("SELECT id FROM roles WHERE name = ? LIMIT 1", r.Name).Scan(&id)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if id == "" {
			id = utils.NewULID()
			_, err = db.Exec(
				"INSERT INTO roles (id, name, display_name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
				id, r.Name, r.DisplayName, r.Description, now, now,
			)
			if err != nil {
				return nil, err
			}
		}
		roleMap[r.Name] = id
	}

	// 2. Permissions
	permissions := []permissionSeed{
		{Name: "users.read", Description: "View users"},
		{Name: "users.create", Description: "Create users"},
		{Name: "users.update", Description: "Update users"},
		{Name: "users.delete", Description: "Delete users"},
		{Name: "roles.read", Description: "View roles"},
		{Name: "roles.create", Description: "Create roles"},
		{Name: "roles.update", Description: "Update roles"},
		{Name: "roles.delete", Description: "Delete roles"},
		{Name: "permissions.read", Description: "View permissions"},
		{Name: "permissions.create", Description: "Create permissions"},
		{Name: "permissions.update", Description: "Update permissions"},
		{Name: "permissions.delete", Description: "Delete permissions"},
		{Name: "posts.read", Description: "View posts"},
		{Name: "posts.create", Description: "Create posts"},
		{Name: "posts.update", Description: "Update posts"},
		{Name: "posts.delete", Description: "Delete posts"},
		{Name: "categories.manage", Description: "Manage categories"},
	}

	permMap := make(map[string]string) // name -> ID
	for _, p := range permissions {
		var id string
		err := db.QueryRow("SELECT id FROM permissions WHERE name = ? LIMIT 1", p.Name).Scan(&id)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if id == "" {
			id = utils.NewULID()
			_, err = db.Exec(
				"INSERT INTO permissions (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
				id, p.Name, p.Description, now, now,
			)
			if err != nil {
				return nil, err
			}
		}
		permMap[p.Name] = id
	}

	// 3. Assign Permissions to Roles
	// Super Admin gets all permissions
	if superAdminID, ok := roleMap["super_admin"]; ok {
		for _, permID := range permMap {
			_, _ = db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
				superAdminID, permID, now)
		}
	}

	// Admin gets users.read, posts.*, categories.manage
	if adminRoleID, ok := roleMap["admin"]; ok {
		adminPerms := []string{
			"users.read",
			"posts.read", "posts.create", "posts.update", "posts.delete",
			"categories.manage",
		}
		for _, pName := range adminPerms {
			if permID, ok := permMap[pName]; ok {
				_, _ = db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
					adminRoleID, permID, now)
			}
		}
	}

	// User gets posts.read
	if userRoleID, ok := roleMap["user"]; ok {
		if permID, ok := permMap["posts.read"]; ok {
			_, _ = db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
				userRoleID, permID, now)
		}
	}

	slog.Info("RBAC seeding completed successfully.")
	return roleMap, nil
}
