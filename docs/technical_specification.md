# Content Management System — Technical Specification

**Project:** mmdbconnect Personal Website  
**Module:** Blog, Article & Journal  
**Technology:** Go (Golang), MySQL 8.0+  
**Document Type:** Technical Requirements  
**Version:** 1.0

---

## 1. Overview

Sistem Content Management sederhana untuk mengelola dan mempublikasikan konten pada website pribadi mmdbconnect menggunakan Go sebagai backend dan MySQL sebagai database.

Sistem mendukung tiga jenis konten:

| Type      | Description                                       |
| --------- | ------------------------------------------------- |
| `blog`    | Tulisan personal, pengalaman, dan opini           |
| `article` | Artikel teknis, tutorial, dan pembahasan mendalam |
| `journal` | Catatan pengembangan, eksperimen, dan dokumentasi |

Seluruh konten disimpan dalam satu tabel utama `posts` dan dibedakan berdasarkan kolom `type`.

---

## 2. Objectives

- Menyediakan sistem publikasi konten yang sederhana dan terstruktur.
- Mendukung Blog, Article, dan Journal.
- Mendukung kategori dan tags.
- Menyediakan metadata dasar untuk SEO.
- Mendukung draft dan publikasi konten.
- Menggunakan backend Go yang modular dan mudah dipelihara.
- Memanfaatkan MySQL sebagai penyimpanan data utama.

---

## 3. Database Schema

Database menggunakan MySQL 8.0+ dengan storage engine InnoDB dan character set `utf8mb4`.

### 3.1. `posts`

| Column             | Data Type                        | Constraints        | Description         |
| ------------------ | -------------------------------- | ------------------ | ------------------- |
| `id`               | BIGINT UNSIGNED                  | PK, AUTO_INCREMENT | Primary identifier  |
| `type`             | ENUM('blog','article','journal') | NOT NULL           | Jenis konten        |
| `title`            | VARCHAR(255)                     | NOT NULL           | Judul konten        |
| `slug`             | VARCHAR(255)                     | UNIQUE, NOT NULL   | URL identifier      |
| `excerpt`          | TEXT                             | NULLABLE           | Ringkasan konten    |
| `content`          | LONGTEXT                         | NOT NULL           | Isi konten Markdown |
| `cover_image`      | VARCHAR(500)                     | NULLABLE           | Path gambar cover   |
| `category_id`      | BIGINT UNSIGNED                  | FK, NULLABLE       | Referensi kategori  |
| `tags`             | JSON                             | NULLABLE           | Daftar tags         |
| `meta_title`       | VARCHAR(255)                     | NULLABLE           | SEO title           |
| `meta_description` | TEXT                             | NULLABLE           | SEO description     |
| `is_featured`      | BOOLEAN                          | DEFAULT FALSE      | Konten unggulan     |
| `status`           | ENUM('draft','published')        | DEFAULT 'draft'    | Status publikasi    |
| `published_at`     | DATETIME                         | NULLABLE           | Tanggal publikasi   |
| `created_at`       | DATETIME                         | NOT NULL           | Tanggal dibuat      |
| `updated_at`       | DATETIME                         | NOT NULL           | Tanggal diperbarui  |

**Constraints:**

- `PRIMARY KEY (id)`
- `UNIQUE (slug)`
- `FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL`
- `CHECK (status <> 'published' OR published_at IS NOT NULL)`

**Indexes:**

- `INDEX (type, status, published_at)`
- `INDEX (status, published_at)`
- `INDEX (category_id)`

### 3.2. `categories`

| Column       | Data Type       | Constraints        | Description        |
| ------------ | --------------- | ------------------ | ------------------ |
| `id`         | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Primary identifier |
| `name`       | VARCHAR(100)    | NOT NULL           | Nama kategori      |
| `slug`       | VARCHAR(120)    | UNIQUE, NOT NULL   | URL identifier     |
| `created_at` | DATETIME        | NOT NULL           | Tanggal dibuat     |
| `updated_at` | DATETIME        | NOT NULL           | Tanggal diperbarui |

### 3.3. `post_views` (Optional)

| Column       | Data Type       | Constraints        | Description         |
| ------------ | --------------- | ------------------ | ------------------- |
| `id`         | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Primary identifier  |
| `post_id`    | BIGINT UNSIGNED | FK, UNIQUE         | Referensi post      |
| `views`      | BIGINT UNSIGNED | DEFAULT 0          | Total kunjungan     |
| `updated_at` | DATETIME        | NOT NULL           | Terakhir diperbarui |

### 3.4. `users`

| Column          | Data Type       | Constraints        | Description              |
| --------------- | --------------- | ------------------ | ------------------------ |
| `id`            | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Primary identifier       |
| `name`          | VARCHAR(100)    | NOT NULL           | Nama administrator       |
| `email`         | VARCHAR(255)    | UNIQUE, NOT NULL   | Email login              |
| `password_hash` | VARCHAR(255)    | NOT NULL           | Password hash (Argon2id) |
| `created_at`    | DATETIME        | NOT NULL           | Tanggal dibuat           |
| `updated_at`    | DATETIME        | NOT NULL           | Tanggal diperbarui       |

### 3.5. `sessions`

| Column       | Data Type       | Constraints                                 | Description                                      |
| ------------ | --------------- | ------------------------------------------- | ------------------------------------------------ |
| `id`         | VARCHAR(64)     | PK                                          | Cryptographic random session token (32-byte hex) |
| `user_id`    | BIGINT UNSIGNED | FK (`users.id` ON DELETE CASCADE), NOT NULL | Referensi administrator                          |
| `expires_at` | DATETIME        | INDEX, NOT NULL                             | Waktu kadaluarsa sesi (7 hari sejak login)       |
| `created_at` | DATETIME        | NOT NULL                                    | Tanggal dibuat                                   |

---

## 4. API Endpoints

### Health & Static Assets

- `GET /health` (Health check server `{"status":"ok"}`)
- `GET /uploads/{file}` (Penyajian file gambar cover yang diupload)

### Public Endpoints

- `GET /api/v1/posts` (query: `type`, `tag`, `featured`, `page`, `limit`)
- `GET /api/v1/posts/{slug}` (detail post, safe markdown html, SEO metadata, increment views)
- `GET /api/v1/categories` (daftar kategori)
- `GET /api/v1/categories/{slug}/posts` (daftar post per kategori; query: `type`, `tag`, `featured`, `page`, `limit`)
- `GET /blog/{slug}`, `GET /articles/{slug}`, `GET /journal/{slug}` (alias URL publik detail post)
- `GET /categories/{slug}` (alias URL publik daftar post per kategori)

### Admin Endpoints

- `POST /api/v1/admin/login` (Admin login via Argon2id & HTTP-only session cookie; rate limited 10 req/menit per IP)
- `POST /api/v1/admin/logout` (Admin logout & hapus session)
- `GET /api/v1/admin/me` (Data admin login)
- `GET /api/v1/admin/posts` (Semua konten draft & published; query: `type`, `status`, `tag`, `featured`, `category_id`, `page`, `limit`)
- `POST /api/v1/admin/posts` (Buat konten baru)
- `GET /api/v1/admin/posts/{id}` (Detail konten)
- `PUT /api/v1/admin/posts/{id}` (Perbarui konten)
- `DELETE /api/v1/admin/posts/{id}` (Hapus konten)
- `GET /api/v1/admin/categories` (Daftar kategori)
- `POST /api/v1/admin/categories` (Buat kategori)
- `PUT /api/v1/admin/categories/{id}` (Perbarui kategori)
- `DELETE /api/v1/admin/categories/{id}` (Hapus kategori)
- `POST /api/v1/admin/upload` (Upload cover image ke disk lokal dengan validasi ekstensi & magic bytes)
