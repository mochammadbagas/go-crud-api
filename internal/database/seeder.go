package database

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

// Seed populates the database with initial admin, roles, permissions, categories, tags, and sample articles.
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

	// 3. Seed Categories, Tags & Articles if empty
	if categoryCount == 0 {
		slog.Info("Seeding initial categories, tags, and sample articles with ULID...")

		cat1ID := utils.NewULID()
		_, err := db.Exec("INSERT INTO categories (id, name, slug, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			cat1ID, "Software Engineering", "software-engineering", "System design, distributed systems, and best practices.", now, now)
		if err != nil {
			return err
		}

		cat2ID := utils.NewULID()
		_, err = db.Exec("INSERT INTO categories (id, name, slug, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			cat2ID, "Tutorials & Tech", "tutorials-tech", "Step-by-step programming guides and tutorials.", now, now)
		if err != nil {
			return err
		}

		cat3ID := utils.NewULID()
		_, err = db.Exec("INSERT INTO categories (id, name, slug, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			cat3ID, "Personal Journey", "personal-journey", "Reflections, career advice, and engineering notes.", now, now)
		if err != nil {
			return err
		}

		// Seed Sample Tags
		tagMap := make(map[string]string)
		initialTags := []string{"Go", "MySQL", "REST API", "Backend", "Career", "Performance"}
		for _, tName := range initialTags {
			tID := utils.NewULID()
			tSlug := utils.GenerateSlug(tName)
			_, _ = db.Exec("INSERT INTO tags (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
				tID, tName, tSlug, now, now)
			tagMap[tName] = tID
		}

		// Sample Published Article 1
		art1ID := utils.NewULID()
		_, err = db.Exec(`
			INSERT INTO articles (
				id, author_id, category_id, title, slug, excerpt, content,
				status, is_featured, published_at, meta_title, meta_description,
				robots_index, robots_follow, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			art1ID,
			adminID,
			cat2ID,
			"Building High Performance REST APIs with Go",
			"building-high-performance-rest-apis-with-go",
			"Comprehensive guide to designing and building high-performance RESTful APIs using Go, Chi, and MySQL.",
			"<h1>Building High Performance REST APIs with Go</h1><p>Go is a highly efficient language for building modern cloud services...</p><pre><code>package main\n\nfunc main() {\n    println(\"Hello, Go!\")\n}</code></pre>",
			"published",
			true,
			now,
			"Building REST APIs with Go - Complete Guide",
			"In-depth tutorial on building REST APIs with Go, MySQL, and clean modular architecture.",
			true,
			true,
			now,
			now,
		)
		if err == nil {
			for _, t := range []string{"Go", "REST API", "Backend"} {
				if tID, ok := tagMap[t]; ok {
					_, _ = db.Exec("INSERT INTO article_tags (article_id, tag_id, created_at) VALUES (?, ?, ?)", art1ID, tID, now)
				}
			}
		}

		// Sample Published Article 2
		art2ID := utils.NewULID()
		_, err = db.Exec(`
			INSERT INTO articles (
				id, author_id, category_id, title, slug, excerpt, content,
				status, is_featured, published_at, meta_title, meta_description,
				robots_index, robots_follow, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			art2ID,
			adminID,
			cat3ID,
			"My Software Development Journey",
			"my-software-development-journey",
			"Reflections on building backend architectures and products over the past decade.",
			"<h1>My Software Development Journey</h1><p>Being a software engineer is a continuous learning and growing journey...</p>",
			"published",
			false,
			now,
			"My Software Development Journey - Reflections",
			"Reflections on building systems and products.",
			true,
			true,
			now,
			now,
		)
		if err == nil {
			if tID, ok := tagMap["Career"]; ok {
				_, _ = db.Exec("INSERT INTO article_tags (article_id, tag_id, created_at) VALUES (?, ?, ?)", art2ID, tID, now)
			}
		}

		// Sample Draft Article
		art3ID := utils.NewULID()
		_, _ = db.Exec(`
			INSERT INTO articles (
				id, author_id, category_id, title, slug, excerpt, content,
				status, is_featured, published_at, meta_title, meta_description,
				robots_index, robots_follow, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			art3ID,
			adminID,
			cat1ID,
			"Upcoming: Event Driven Architecture in Go",
			"upcoming-event-driven-architecture-in-go",
			"Draft article exploring message queues, streaming patterns, and idempotency.",
			"<h1>Event Driven Architecture</h1><p>This article is currently in draft status...</p>",
			"draft",
			false,
			nil,
			nil,
			nil,
			true,
			true,
			now,
			now,
		)

		slog.Info("Initial categories, tags, and sample articles seeded successfully.")
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
		{Name: "admin", DisplayName: "Admin", Description: "Manages users, content, categories, tags, and media"},
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
		{Name: "articles.read", Description: "View articles"},
		{Name: "articles.create", Description: "Create articles"},
		{Name: "articles.update", Description: "Update articles"},
		{Name: "articles.delete", Description: "Delete articles"},
		{Name: "categories.manage", Description: "Manage categories"},
		{Name: "tags.read", Description: "View tags"},
		{Name: "tags.create", Description: "Create tags"},
		{Name: "tags.update", Description: "Update tags"},
		{Name: "tags.delete", Description: "Delete tags"},
		{Name: "media.read", Description: "View media library"},
		{Name: "media.upload", Description: "Upload media files"},
		{Name: "media.delete", Description: "Delete media files"},
		{Name: "analytics.read", Description: "View analytics reports"},
		{Name: "settings.manage", Description: "Manage system settings"},
		// Legacy aliases for backward compatibility
		{Name: "posts.read", Description: "Legacy view posts alias"},
		{Name: "posts.create", Description: "Legacy create posts alias"},
		{Name: "posts.update", Description: "Legacy update posts alias"},
		{Name: "posts.delete", Description: "Legacy delete posts alias"},
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

	// Admin gets users.read, articles.*, categories.manage, tags.*, media.*, analytics.read, settings.manage
	if adminRoleID, ok := roleMap["admin"]; ok {
		adminPerms := []string{
			"users.read",
			"articles.read", "articles.create", "articles.update", "articles.delete",
			"posts.read", "posts.create", "posts.update", "posts.delete",
			"categories.manage",
			"tags.read", "tags.create", "tags.update", "tags.delete",
			"media.read", "media.upload", "media.delete",
			"analytics.read",
			"settings.manage",
		}
		for _, pName := range adminPerms {
			if permID, ok := permMap[pName]; ok {
				_, _ = db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
					adminRoleID, permID, now)
			}
		}
	}

	// User gets articles.read
	if userRoleID, ok := roleMap["user"]; ok {
		for _, pName := range []string{"articles.read", "posts.read"} {
			if permID, ok := permMap[pName]; ok {
				_, _ = db.Exec("INSERT IGNORE INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
					userRoleID, permID, now)
			}
		}
	}

	slog.Info("RBAC seeding completed successfully.")
	return roleMap, nil
}
