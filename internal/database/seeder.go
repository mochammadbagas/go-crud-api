package database

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"

	"github.com/alexedwards/argon2id"
)

// Seed populates the database with initial admin, categories, and sample posts if empty.
func Seed(db *sql.DB, cfg *config.Config) error {
	var userCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return err
	}

	now := time.Now().UTC()

	// 1. Seed Administrator if none exists
	if userCount == 0 {
		slog.Info("Menyemai data administrator awal...")
		passwordHash, err := argon2id.CreateHash(cfg.AdminPassword, argon2id.DefaultParams)
		if err != nil {
			return err
		}

		_, err = db.Exec(
			"INSERT INTO users (name, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			cfg.AdminName, cfg.AdminEmail, passwordHash, now, now,
		)
		if err != nil {
			return err
		}
		slog.Info("Administrator berhasil dibuat", slog.String("email", cfg.AdminEmail))
	}

	var categoryCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount); err != nil {
		return err
	}

	// 2. Seed Categories & Posts if empty
	if categoryCount == 0 {
		slog.Info("Menyemai data awal kategori dan postingan...")

		// Categories
		res1, err := db.Exec("INSERT INTO categories (name, slug, created_at, updated_at) VALUES (?, ?, ?, ?)",
			"Software Engineering", "software-engineering", now, now)
		if err != nil {
			return err
		}
		cat1ID, _ := res1.LastInsertId()

		res2, err := db.Exec("INSERT INTO categories (name, slug, created_at, updated_at) VALUES (?, ?, ?, ?)",
			"Tutorials & Tech", "tutorials-tech", now, now)
		if err != nil {
			return err
		}
		cat2ID, _ := res2.LastInsertId()

		res3, err := db.Exec("INSERT INTO categories (name, slug, created_at, updated_at) VALUES (?, ?, ?, ?)",
			"Personal Journey", "personal-journey", now, now)
		if err != nil {
			return err
		}
		cat3ID, _ := res3.LastInsertId()

		// Sample Posts
		tags1, _ := json.Marshal([]string{"Go", "MySQL", "REST API", "Backend"})
		tags2, _ := json.Marshal([]string{"Career", "Reflection", "Growth"})
		tags3, _ := json.Marshal([]string{"Go", "Rust", "Benchmarking", "Performance"})

		// Article
		_, err = db.Exec(`
			INSERT INTO posts (
				type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
				type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
				type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
				type, title, slug, excerpt, content, cover_image, category_id, tags,
				meta_title, meta_description, is_featured, status, published_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
