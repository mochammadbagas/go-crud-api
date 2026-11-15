# Content Management System — Technical Specification

**Project:** Go REST API Starter Kit  
**Module:** Core Auth, Account Recovery, Pluggable Mailer, Granular RBAC, Admin User Management, & Content Modules  
**Technology:** Go (Golang), MySQL 8.0+, ULID, JWT HS256, Pluggable Mailer (Log & Resend), RBAC Middleware  
**Document Type:** Technical Requirements & Schema Specification  
**Version:** 2.3 (Completed: Pluggable Storage Local & Cloudflare R2 / AWS S3)

---

## 1. Overview

Sistem backend RESTful API modular yang menggabungkan sistem autentikasi modern (JWT + Refresh Token Rotation), pemulihan akun & verifikasi email dengan arsitektur _pluggable mailer_ (LogMailer & Resend), arsitektur _pluggable storage_ (LocalStorage & Cloudflare R2 / AWS S3), otorisasi berbasis hak akses (_Role-Based Access Control_ - RBAC) dengan format izin `resource.action`, modul manajemen pengguna administrator, dan modul konten publikasi terstruktur (Blog, Article, dan Journal) menggunakan identitas berbasis **ULID (Crockford Base32 26-karakter)**.

Sistem konten mendukung tiga jenis publikasi:

| Type      | Description                                       |
| :-------- | :------------------------------------------------ |
| `blog`    | Tulisan personal, pengalaman, dan opini           |
| `article` | Artikel teknis, tutorial, dan pembahasan mendalam |
| `journal` | Catatan pengembangan, eksperimen, dan dokumentasi |

Seluruh konten disimpan dalam satu tabel utama `posts` dan dibedakan berdasarkan kolom `type`.

---

## 2. Objectives

- Menyediakan identitas terdesentralisasi bebas enumerasi menggunakan **ULID** (`CHAR(26)`).
- Menyediakan sistem autentikasi stateless **JWT Access Token** dan stateful **Refresh Token Rotation (RTR)** dengan deteksi pembajakan (_Reuse Detection_).
- Menyediakan alur pemulihan akun dan verifikasi email yang aman:
  - **Forgot & Reset Password**: Token acak SHA-256 (TTL 15 menit), anti user-enumeration, dan pencabutan seluruh sesi aktif.
  - **Email Verification**: Verifikasi otomatis pendaftaran akun dan pengiriman ulang (_resend verification_).
  - **Email Change Verification**: Penggantian alamat email dengan verifikasi password lama dan konfirmasi token di email baru.
- Menyediakan arsitektur **Pluggable Mailer** berbasis driver: `LogMailer` untuk development lokal 100% tanpa konfigurasi/kredensial, dan `ResendMailer` untuk production siap pakai.
- Menyediakan arsitektur **Pluggable Storage** berbasis driver: `LocalStorage` untuk development lokal (folder `./uploads`) dan `S3Storage` untuk Cloudflare R2 / AWS S3 / MinIO siap pakai, lengkap dengan penamaan file baru 100% menggunakan ULID murni (`CHAR(26)`) serta **mekanisme Auto-Cleanup** (otomatis membersihkan file lama dari storage saat pergantian avatar, penghapusan avatar, update/delete cover artikel, dan penghapusan pengguna).
- Menyediakan otorisasi granular **Role-Based Access Control (RBAC)**:
  - Format izin standar `resource.action` (contoh: `users.read`, `roles.manage`).
  - Dukungan multi-role per pengguna (_user can have multiple roles_).
  - _Super Admin Wildcard Bypass (`_`)\* pada permission middleware.
  - Proteksi sistem terhadap penghapusan/penonaktifan Super Admin terakhir.
- Menyediakan modul **Admin User Management** (pencarian, filter status, filter role, pagination, mutasi role, dan pembekuan akun).
- Menyediakan modul **User Management** (profil personal, avatar upload/delete, ganti username, dan status akun).
- Menyediakan sistem publikasi konten yang sederhana dan terstruktur (kategori, tags JSON, SEO fallback, dan status draft/published).
- Memastikan seluruh query database type-safe menggunakan **SQLC**.

---

## 3. Database Schema (MySQL 8.0+)

Database menggunakan MySQL 8.0+ dengan storage engine InnoDB dan character set `utf8mb4`.

### 3.1. `users`

| Column              | Data Type                                                    | Constraints                | Description                                |
| :------------------ | :----------------------------------------------------------- | :------------------------- | :----------------------------------------- |
| `id`                | CHAR(26)                                                     | PK                         | Primary identifier (ULID)                  |
| `username`          | VARCHAR(50)                                                  | UNIQUE, NOT NULL           | Nama unik pengguna (3-30 karakter)         |
| `email`             | VARCHAR(255)                                                 | UNIQUE, NOT NULL           | Alamat email login                         |
| `password_hash`     | VARCHAR(255)                                                 | NOT NULL                   | Hash password Argon2id                     |
| `role`              | ENUM('admin','user')                                         | NOT NULL, DEFAULT 'user'   | Kolom transisi (dimigrasi ke `user_roles`) |
| `status`            | ENUM('active','inactive','pending_verification','suspended') | NOT NULL, DEFAULT 'active' | Status aktif akun pengguna                 |
| `email_verified_at` | DATETIME                                                     | NULLABLE                   | Waktu verifikasi email                     |
| `created_at`        | DATETIME                                                     | NOT NULL                   | Waktu dibuat                               |
| `updated_at`        | DATETIME                                                     | NOT NULL                   | Waktu diperbarui                           |

### 3.2. `user_profiles`

| Column         | Data Type    | Constraints                    | Description             |
| :------------- | :----------- | :----------------------------- | :---------------------- |
| `user_id`      | CHAR(26)     | PK, FK -> `users.id` (CASCADE) | Referensi akun pengguna |
| `full_name`    | VARCHAR(100) | NOT NULL, DEFAULT ''           | Nama lengkap pengguna   |
| `bio`          | TEXT         | NULLABLE                       | Bio singkat             |
| `phone_number` | VARCHAR(20)  | NULLABLE                       | Nomor telepon           |
| `avatar_url`   | VARCHAR(500) | NULLABLE                       | URL/Path foto avatar    |
| `created_at`   | DATETIME     | NOT NULL                       | Waktu dibuat            |
| `updated_at`   | DATETIME     | NOT NULL                       | Waktu diperbarui        |

### 3.3. `refresh_tokens`

| Column         | Data Type    | Constraints                | Description                     |
| :------------- | :----------- | :------------------------- | :------------------------------ |
| `id`           | CHAR(26)     | PK                         | Primary identifier (ULID)       |
| `user_id`      | CHAR(26)     | FK -> `users.id` (CASCADE) | Referensi akun pengguna         |
| `token_hash`   | CHAR(64)     | INDEX, NOT NULL            | SHA-256 hash dari refresh token |
| `token_family` | VARCHAR(64)  | INDEX, NOT NULL            | Identitas silsilah rotasi token |
| `is_revoked`   | BOOLEAN      | DEFAULT FALSE, NOT NULL    | Status pencabutan token         |
| `expires_at`   | DATETIME     | INDEX, NOT NULL            | Waktu kedaluwarsa token         |
| `created_at`   | DATETIME     | NOT NULL                   | Waktu dibuat                    |
| `user_agent`   | VARCHAR(255) | NULLABLE                   | Informasi browser/client        |
| `ip_address`   | VARCHAR(45)  | NULLABLE                   | Alamat IP pembuat token         |

### 3.4. `auth_tokens`

| Column       | Data Type                                                    | Constraints                | Description                                    |
| :----------- | :----------------------------------------------------------- | :------------------------- | :--------------------------------------------- |
| `id`         | CHAR(26)                                                     | PK                         | Primary identifier (ULID)                      |
| `user_id`    | CHAR(26)                                                     | FK -> `users.id` (CASCADE) | Referensi akun pengguna                        |
| `token_hash` | CHAR(64)                                                     | INDEX, NOT NULL            | SHA-256 hash dari token sementara              |
| `token_type` | ENUM('password_reset', 'email_verification', 'email_change') | NOT NULL                   | Tipe kegunaan token                            |
| `payload`    | VARCHAR(255)                                                 | NULLABLE                   | Data kontekstual tambahan (misal: `new_email`) |
| `is_used`    | BOOLEAN                                                      | DEFAULT FALSE, NOT NULL    | Status penggunaan token                        |
| `expires_at` | DATETIME                                                     | INDEX, NOT NULL            | Waktu kedaluwarsa token                        |
| `created_at` | DATETIME                                                     | NOT NULL                   | Waktu dibuat                                   |

### 3.5. `roles` (Master Role RBAC)

| Column         | Data Type    | Constraints      | Description                                    |
| :------------- | :----------- | :--------------- | :--------------------------------------------- |
| `id`           | CHAR(26)     | PK               | Primary identifier (ULID)                      |
| `name`         | VARCHAR(50)  | UNIQUE, NOT NULL | Nama unik role: `super_admin`, `admin`, `user` |
| `display_name` | VARCHAR(100) | NOT NULL         | Label tampilan role: 'Super Administrator'     |
| `description`  | TEXT         | NULLABLE         | Deskripsi tugas dan wewenang role              |
| `created_at`   | DATETIME     | NOT NULL         | Waktu dibuat                                   |
| `updated_at`   | DATETIME     | NOT NULL         | Waktu diperbarui                               |

### 3.6. `permissions` (Master Permission RBAC)

| Column        | Data Type    | Constraints      | Description                            |
| :------------ | :----------- | :--------------- | :------------------------------------- |
| `id`          | CHAR(26)     | PK               | Primary identifier (ULID)              |
| `name`        | VARCHAR(100) | UNIQUE, NOT NULL | Format `resource.action`: `users.read` |
| `description` | TEXT         | NULLABLE         | Penjelasan fungsional izin             |
| `created_at`  | DATETIME     | NOT NULL         | Waktu dibuat                           |
| `updated_at`  | DATETIME     | NOT NULL         | Waktu diperbarui                       |

### 3.7. `role_permissions` (Pivot Role & Permission)

| Column          | Data Type | Constraints                          | Description                 |
| :-------------- | :-------- | :----------------------------------- | :-------------------------- |
| `role_id`       | CHAR(26)  | Composite PK, FK -> `roles.id`       | Referensi master role       |
| `permission_id` | CHAR(26)  | Composite PK, FK -> `permissions.id` | Referensi master permission |
| `created_at`    | DATETIME  | NOT NULL                             | Waktu penetapan izin        |

### 3.8. `user_roles` (Pivot User & Role / Multi-Role)

| Column       | Data Type | Constraints                    | Description             |
| :----------- | :-------- | :----------------------------- | :---------------------- |
| `user_id`    | CHAR(26)  | Composite PK, FK -> `users.id` | Referensi akun pengguna |
| `role_id`    | CHAR(26)  | Composite PK, FK -> `roles.id` | Referensi master role   |
| `created_at` | DATETIME  | NOT NULL                       | Waktu penetapan role    |

### 3.9. `posts`

| Column             | Data Type                        | Constraints                      | Description               |
| :----------------- | :------------------------------- | :------------------------------- | :------------------------ |
| `id`               | CHAR(26)                         | PK                               | Primary identifier (ULID) |
| `type`             | ENUM('blog','article','journal') | NOT NULL                         | Jenis konten              |
| `title`            | VARCHAR(255)                     | NOT NULL                         | Judul konten              |
| `slug`             | VARCHAR(255)                     | UNIQUE, NOT NULL                 | URL slug identifier       |
| `excerpt`          | TEXT                             | NULLABLE                         | Ringkasan konten          |
| `content`          | LONGTEXT                         | NOT NULL                         | Isi konten Markdown       |
| `cover_image`      | VARCHAR(500)                     | NULLABLE                         | Path gambar cover         |
| `category_id`      | CHAR(26)                         | FK -> `categories.id` (SET NULL) | Referensi kategori (ULID) |
| `tags`             | JSON                             | NULLABLE                         | Daftar tags JSON array    |
| `meta_title`       | VARCHAR(255)                     | NULLABLE                         | SEO title                 |
| `meta_description` | TEXT                             | NULLABLE                         | SEO description           |
| `is_featured`      | BOOLEAN                          | DEFAULT FALSE                    | Konten unggulan           |
| `status`           | ENUM('draft','published')        | DEFAULT 'draft'                  | Status publikasi          |
| `published_at`     | DATETIME                         | NULLABLE                         | Tanggal publikasi         |
| `created_at`       | DATETIME                         | NOT NULL                         | Tanggal dibuat            |
| `updated_at`       | DATETIME                         | NOT NULL                         | Tanggal diperbarui        |

### 3.10. `categories`

| Column       | Data Type    | Constraints      | Description               |
| :----------- | :----------- | :--------------- | :------------------------ |
| `id`         | CHAR(26)     | PK               | Primary identifier (ULID) |
| `name`       | VARCHAR(100) | NOT NULL         | Nama kategori             |
| `slug`       | VARCHAR(120) | UNIQUE, NOT NULL | URL identifier            |
| `created_at` | DATETIME     | NOT NULL         | Tanggal dibuat            |
| `updated_at` | DATETIME     | NOT NULL         | Tanggal diperbarui        |

### 3.11. `post_views`

| Column       | Data Type       | Constraints                        | Description               |
| :----------- | :-------------- | :--------------------------------- | :------------------------ |
| `id`         | CHAR(26)        | PK                                 | Primary identifier (ULID) |
| `post_id`    | CHAR(26)        | FK -> `posts.id` (CASCADE), UNIQUE | Referensi post (ULID)     |
| `views`      | BIGINT UNSIGNED | DEFAULT 0                          | Total kunjungan           |
| `updated_at` | DATETIME        | NOT NULL                           | Terakhir diperbarui       |

---

## 4. API Endpoints

### 4.1. Authentication (`/api/v1/auth`)

- `POST /api/v1/auth/register` (Public, Rate Limited, Auto Verification Email)
- `POST /api/v1/auth/login` (Public, Rate Limited, Hybrid Token Delivery)
- `POST /api/v1/auth/refresh` (Public/Cookie, Refresh Token Rotation & Reuse Detection)
- `POST /api/v1/auth/logout` (Protected Bearer, Revoke Refresh Token & Clear Cookie)
- `POST /api/v1/auth/change-password` (Protected Bearer, Change Password & Invalidate Sessions)
- `POST /api/v1/auth/forgot-password` (Public, Rate Limited, Anti User-Enumeration)
- `POST /api/v1/auth/reset-password` (Public, Rate Limited, Argon2id Hash, Revoke All Sessions)
- `GET` & `POST /api/v1/auth/verify-email` (Public, Query Param `?token=` atau JSON Body)
- `POST /api/v1/auth/resend-verification` (Public, Rate Limited, Anti User-Enumeration)

### 4.2. User Management (`/api/v1/users`)

- `GET /api/v1/users/me` (Protected Bearer, Profil Akun & Personal)
- `PUT /api/v1/users/profile` (Protected Bearer, Update Nama, Bio, Telepon)
- `POST /api/v1/users/avatar` (Protected Bearer, Upload Avatar Multipart JPG/PNG/WEBP)
- `DELETE /api/v1/users/avatar` (Protected Bearer, Hapus Avatar)
- `PUT /api/v1/users/username` (Protected Bearer, Ganti Username Unik)
- `GET /api/v1/users/status` (Protected Bearer, Cek Status Akun & Status Verifikasi Email)
- `POST /api/v1/users/email/request-change` (Protected Bearer, Validasi Password Lama & Send Verification to New Email)
- `GET` & `POST /api/v1/users/email/confirm-change` (Public, Query Param `?token=` atau JSON Body, Update Email & Mark Verified)

### 4.3. Role & Permission Management (`/api/v1/admin/roles` & `/permissions`)

- `GET /api/v1/admin/roles` (`roles.read`)
- `POST /api/v1/admin/roles` (`roles.create`)
- `GET /api/v1/admin/roles/{id}` (`roles.read`)
- `PUT /api/v1/admin/roles/{id}` (`roles.update`)
- `DELETE /api/v1/admin/roles/{id}` (`roles.delete`, Super Admin role dilindungi)
- `PUT /api/v1/admin/roles/{id}/permissions` (`roles.update`, Sinkronisasi daftar izin role)
- `GET /api/v1/admin/permissions` (`permissions.read`)
- `POST /api/v1/admin/permissions` (`permissions.create`)
- `PUT /api/v1/admin/permissions/{id}` (`permissions.update`)
- `DELETE /api/v1/admin/permissions/{id}` (`permissions.delete`)

### 4.4. Admin User Management (`/api/v1/admin/users`)

- `GET /api/v1/admin/users` (`users.read`, Search, Filter Status & Role, Pagination)
- `GET /api/v1/admin/users/{id}` (`users.read`, Detail Akun, Profil, Roles)
- `PUT /api/v1/admin/users/{id}` (`users.update`, Perbarui Informasi User)
- `PUT /api/v1/admin/users/{id}/status` (`users.update`, Ubah status `active`/`inactive`/`suspended`)
- `PUT /api/v1/admin/users/{id}/roles` (`users.update`, Tetapkan multi-role)
- `DELETE /api/v1/admin/users/{id}` (`users.delete`, Dilindungi safeguard Super Admin terakhir)

### 4.5. Admin Content Management (`/api/v1/admin/posts` & `/categories`)

- `GET /api/v1/admin/posts` & `POST /api/v1/admin/posts` (`posts.create`)
- `GET /api/v1/admin/posts/{id}` & `PUT /api/v1/admin/posts/{id}` & `DELETE /api/v1/admin/posts/{id}` (`posts.update` / `delete`)
- `GET /api/v1/admin/categories` & `POST /api/v1/admin/categories` (`categories.manage`)
- `PUT /api/v1/admin/categories/{id}` & `DELETE /api/v1/admin/categories/{id}` (`categories.manage`)
- `POST /api/v1/admin/upload` (Upload Cover Image)

### 4.6. Public Content (`/api/v1/posts` & `/api/v1/categories`)

- `GET /api/v1/posts` (Filter: `type`, `tag`, `featured`, `page`, `limit`)
- `GET /api/v1/posts/{slug}` (Detail post published & increment views)
- `GET /api/v1/categories` & `GET /api/v1/categories/{slug}/posts`
- Aliases: `/blog/{slug}`, `/articles/{slug}`, `/journal/{slug}`, `/categories/{slug}`
