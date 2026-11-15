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
		slog.Info("Menyemai data administrator awal...")
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
			adminID, cfg.AdminName, "Administrator sistem", now, now,
		)
		if err != nil {
			return err
		}

		slog.Info("Administrator berhasil dibuat", slog.String("email", cfg.AdminEmail), slog.String("id", adminID))
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
		slog.Info("Menyemai data awal kategori dan postingan dengan ULID...")

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
			"Panduan komprehensif merancang dan membangun RESTful API performa tinggi menggunakan Go dan Chi.",
			"# Building REST APIs with Go\n\nGo adalah bahasa yang sangat efisien untuk membangun layanan web modern...\n\n```go\npackage main\n\nfunc main() {\n    println(\"Hello, CMS!\")\n}\n```",
			nil,
			cat2ID,
			tags1,
			"Building REST APIs with Go - Complete Guide",
			"Tutorial mendalam membangun REST API dengan Go, MySQL, dan arsitektur modular yang rapi.",
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
			"Refleksi perjalanan pengalaman membangun produk dan sistem backend selama bertahun-tahun.",
			"# My Development Journey\n\nMenjadi seorang software engineer adalah perjalanan belajar tanpa akhir...",
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
			"Catatan eksperimen perbandingan performa hashing file berukuran besar antara Rust dan Go.",
			"# Rust vs Go File Hashing\n\nEksperimen hari ini menguji throughput I/O dan utilisasi CPU...",
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
			"Draft tulisan yang mengeksplorasi message queue dan event streaming.",
			"# Event Driven Architecture\n\nKonten ini masih dalam status draft...",
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

		slog.Info("Seeder awal kategori dan postingan berhasil diselesaikan.")
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
	slog.Info("Menyemai data awal RBAC (Roles & Permissions)...")

	// 1. Roles
	roles := []roleSeed{
		{Name: "super_admin", DisplayName: "Super Admin", Description: "Memiliki akses administratif tertinggi"},
		{Name: "admin", DisplayName: "Admin", Description: "Mengelola pengguna dan fitur administratif sesuai permission"},
		{Name: "user", DisplayName: "User", Description: "Pengguna standar aplikasi"},
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
		{Name: "users.read", Description: "Melihat pengguna"},
		{Name: "users.create", Description: "Membuat pengguna"},
		{Name: "users.update", Description: "Memperbarui pengguna"},
		{Name: "users.delete", Description: "Menghapus pengguna"},
		{Name: "roles.read", Description: "Melihat role"},
		{Name: "roles.create", Description: "Membuat role"},
		{Name: "roles.update", Description: "Memperbarui role"},
		{Name: "roles.delete", Description: "Menghapus role"},
		{Name: "permissions.read", Description: "Melihat permission"},
		{Name: "permissions.create", Description: "Membuat permission"},
		{Name: "permissions.update", Description: "Memperbarui permission"},
		{Name: "permissions.delete", Description: "Menghapus permission"},
		{Name: "posts.read", Description: "Melihat postingan"},
		{Name: "posts.create", Description: "Membuat postingan"},
		{Name: "posts.update", Description: "Memperbarui postingan"},
		{Name: "posts.delete", Description: "Menghapus postingan"},
		{Name: "categories.manage", Description: "Mengelola kategori"},
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

	slog.Info("Seeder RBAC berhasil diselesaikan.")
	return roleMap, nil
}
