# Content Management System — Technical Specification

**Project:** Go REST API Starter Kit  
**Module:** Core Auth, User Management, & Content Modules (Blog, Article & Journal)  
**Technology:** Go (Golang), MySQL 8.0+, ULID, JWT HS256  
**Document Type:** Technical Requirements & Schema Specification  
**Version:** 2.0 (Starter Kit Upgrade)  

---

## 1. Overview

Sistem backend RESTful API modular yang menggabungkan sistem autentikasi modern (JWT + Refresh Token Rotation), manajemen profil pengguna, dan modul konten publikasi terstruktur (Blog, Article, dan Journal) menggunakan identitas berbasis **ULID (Crockford Base32 26-karakter)**.

Sistem konten mendukung tiga jenis publikasi:

| Type | Description |
| :--- | :--- |
| `blog` | Tulisan personal, pengalaman, dan opini |
| `article` | Artikel teknis, tutorial, dan pembahasan mendalam |
| `journal` | Catatan pengembangan, eksperimen, dan dokumentasi |

Seluruh konten disimpan dalam satu tabel utama `posts` dan dibedakan berdasarkan kolom `type`.

---

## 2. Objectives

- Menyediakan identitas terdesentralisasi bebas enumerasi menggunakan **ULID** (`CHAR(26)`).
- Menyediakan sistem autentikasi stateless **JWT Access Token** dan stateful **Refresh Token Rotation (RTR)** dengan deteksi pembajakan (*Reuse Detection*).
- Menyediakan modul **User Management** (profil personal, avatar upload/delete, ganti username, dan status akun).
- Menyediakan sistem publikasi konten yang sederhana dan terstruktur (kategori, tags JSON, SEO fallback, dan status draft/published).
- Memastikan seluruh query database type-safe menggunakan **SQLC**.

---

## 3. Database Schema (MySQL 8.0+)

Database menggunakan MySQL 8.0+ dengan storage engine InnoDB dan character set `utf8mb4`.

### 3.1. `users`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | CHAR(26) | PK | Primary identifier (ULID) |
| `username` | VARCHAR(50) | UNIQUE, NOT NULL | Nama unik pengguna (3-30 karakter) |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL | Alamat email login |
| `password_hash` | VARCHAR(255) | NOT NULL | Hash password Argon2id |
| `role` | ENUM('admin','user') | NOT NULL, DEFAULT 'user' | Role otorisasi |
| `status` | ENUM('active','pending_verification','suspended') | NOT NULL, DEFAULT 'active' | Status aktif akun |
| `email_verified_at` | DATETIME | NULLABLE | Waktu verifikasi email |
| `created_at` | DATETIME | NOT NULL | Waktu dibuat |
| `updated_at` | DATETIME | NOT NULL | Waktu diperbarui |

### 3.2. `user_profiles`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `user_id` | CHAR(26) | PK, FK -> `users.id` (CASCADE) | Referensi akun pengguna |
| `full_name` | VARCHAR(100) | NOT NULL, DEFAULT '' | Nama lengkap pengguna |
| `bio` | TEXT | NULLABLE | Bio singkat |
| `phone_number` | VARCHAR(20) | NULLABLE | Nomor telepon |
| `avatar_url` | VARCHAR(500) | NULLABLE | URL/Path foto avatar |
| `created_at` | DATETIME | NOT NULL | Waktu dibuat |
| `updated_at` | DATETIME | NOT NULL | Waktu diperbarui |

### 3.3. `refresh_tokens`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | CHAR(26) | PK | Primary identifier (ULID) |
| `user_id` | CHAR(26) | FK -> `users.id` (CASCADE) | Referensi akun pengguna |
| `token_hash` | CHAR(64) | INDEX, NOT NULL | SHA-256 hash dari refresh token |
| `token_family` | VARCHAR(64) | INDEX, NOT NULL | Identitas silsilah rotasi token |
| `is_revoked` | BOOLEAN | DEFAULT FALSE, NOT NULL | Status pencabutan token |
| `expires_at` | DATETIME | INDEX, NOT NULL | Waktu kedaluwarsa token |
| `created_at` | DATETIME | NOT NULL | Waktu dibuat |
| `user_agent` | VARCHAR(255) | NULLABLE | Informasi browser/client |
| `ip_address` | VARCHAR(45) | NULLABLE | Alamat IP pembuat token |

### 3.4. `posts`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | CHAR(26) | PK | Primary identifier (ULID) |
| `type` | ENUM('blog','article','journal') | NOT NULL | Jenis konten |
| `title` | VARCHAR(255) | NOT NULL | Judul konten |
| `slug` | VARCHAR(255) | UNIQUE, NOT NULL | URL slug identifier |
| `excerpt` | TEXT | NULLABLE | Ringkasan konten |
| `content` | LONGTEXT | NOT NULL | Isi konten Markdown |
| `cover_image` | VARCHAR(500) | NULLABLE | Path gambar cover |
| `category_id` | CHAR(26) | FK -> `categories.id` (SET NULL) | Referensi kategori (ULID) |
| `tags` | JSON | NULLABLE | Daftar tags JSON array |
| `meta_title` | VARCHAR(255) | NULLABLE | SEO title |
| `meta_description`| TEXT | NULLABLE | SEO description |
| `is_featured` | BOOLEAN | DEFAULT FALSE | Konten unggulan |
| `status` | ENUM('draft','published') | DEFAULT 'draft' | Status publikasi |
| `published_at` | DATETIME | NULLABLE | Tanggal publikasi |
| `created_at` | DATETIME | NOT NULL | Tanggal dibuat |
| `updated_at` | DATETIME | NOT NULL | Tanggal diperbarui |

### 3.5. `categories`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | CHAR(26) | PK | Primary identifier (ULID) |
| `name` | VARCHAR(100) | NOT NULL | Nama kategori |
| `slug` | VARCHAR(120) | UNIQUE, NOT NULL | URL identifier |
| `created_at` | DATETIME | NOT NULL | Tanggal dibuat |
| `updated_at` | DATETIME | NOT NULL | Tanggal diperbarui |

### 3.6. `post_views`

| Column | Data Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | CHAR(26) | PK | Primary identifier (ULID) |
| `post_id` | CHAR(26) | FK -> `posts.id` (CASCADE), UNIQUE | Referensi post (ULID) |
| `views` | BIGINT UNSIGNED | DEFAULT 0 | Total kunjungan |
| `updated_at` | DATETIME | NOT NULL | Terakhir diperbarui |

---

## 4. API Endpoints

### 4.1. Authentication (`/api/v1/auth`)
- `POST /api/v1/auth/register` (Public, Rate Limited)
- `POST /api/v1/auth/login` (Public, Rate Limited, Hybrid Token Delivery)
- `POST /api/v1/auth/refresh` (Public/Cookie, Refresh Token Rotation & Reuse Detection)
- `POST /api/v1/auth/logout` (Protected Bearer, Revoke Refresh Token & Clear Cookie)
- `POST /api/v1/auth/change-password` (Protected Bearer, Change Password & Invalidate Sessions)

### 4.2. User Management (`/api/v1/users`)
- `GET /api/v1/users/me` (Protected Bearer, Profil Akun & Personal)
- `PUT /api/v1/users/profile` (Protected Bearer, Update Nama, Bio, Telepon)
- `POST /api/v1/users/avatar` (Protected Bearer, Upload Avatar Multipart JPG/PNG/WEBP)
- `DELETE /api/v1/users/avatar` (Protected Bearer, Hapus Avatar)
- `PUT /api/v1/users/username` (Protected Bearer, Ganti Username Unik)
- `GET /api/v1/users/status` (Protected Bearer, Cek Status Akun)

### 4.3. Admin Management (`/api/v1/admin`)
- `GET /api/v1/admin/posts` & `POST /api/v1/admin/posts`
- `GET /api/v1/admin/posts/{id}` & `PUT /api/v1/admin/posts/{id}` & `DELETE /api/v1/admin/posts/{id}`
- `GET /api/v1/admin/categories` & `POST /api/v1/admin/categories`
- `PUT /api/v1/admin/categories/{id}` & `DELETE /api/v1/admin/categories/{id}`
- `POST /api/v1/admin/upload` (Upload Cover Image)

### 4.4. Public Content (`/api/v1/posts` & `/api/v1/categories`)
- `GET /api/v1/posts` (Filter: `type`, `tag`, `featured`, `page`, `limit`)
- `GET /api/v1/posts/{slug}` (Detail post published & increment views)
- `GET /api/v1/categories` & `GET /api/v1/categories/{slug}/posts`
- Aliases: `/blog/{slug}`, `/articles/{slug}`, `/journal/{slug}`, `/categories/{slug}`
