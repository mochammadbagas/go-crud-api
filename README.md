# Go REST API Starter Kit

Template backend RESTful API modular, tangguh, dan siap pakai (_production-ready_) berbasis **Go (Golang)** dan **MySQL 8.0+**. Dirancang sebagai pondasi (_starter kit_) untuk berbagai proyek web dan aplikasi backend modern.

---

## 1. Fitur Utama & Keunggulan

- **Identitas Global Berbasis ULID**: Menggunakan _Universally Unique Lexicographically Sortable Identifier_ (`CHAR(26)`) pada seluruh tabel untuk mencegah enumerasi URL, ramah JSON/JavaScript 53-bit, dan otomatis terurut kronologis (`ORDER BY id DESC`).
- **Autentikasi Modern (JWT + Refresh Token Rotation)**:
  - **Access Token**: Stateless JWT (HS256) dengan masa aktif pendek (default 15 menit).
  - **Refresh Token (RTR)**: Stateful token acak 64-karakter yang disimpan dalam bentuk hash **SHA-256** di database.
  - **Reuse Detection (Token Family)**: Mendeteksi indikasi pembajakan sesi saat token lama yang sudah dirotasi dicoba digunakan kembali, otomatis mencabut seluruh sesi terkait.
- **Delivery Token Hybrid**: Mendukung konsumsi client Web/SPA via **HTTP-only Cookie** (`SameSite=Lax`, `Path=/api/v1/auth`) maupun Mobile/Postman via **JSON Response Body** & Header `Authorization: Bearer <token>`.
- **Pluggable Mailer Architecture (LogMailer & Resend)**: Arsitektur email berbasis driver/interface. Di lingkungan lokal/pengembangan, secara default menggunakan `LogMailer` yang mencetak outbox email langsung ke konsol terminal tanpa butuh akun atau jaringan eksternal. Di lingkungan production, langsung siap _plug-and-play_ dengan layanan transactional email modern (Resend) hanya dengan mengatur konfigurasi di `.env`.
- **Alur Pemulihan & Verifikasi Akun Lengkap**:
  - **Forgot Password**: Permintaan reset password dengan proteksi ketat anti user-enumeration (respons 200 konstan).
  - **Reset Password**: Validasi token SHA-256 15 menit, pembaruan password hash Argon2id, dan otomatis mencabut seluruh sesi refresh token aktif pengguna.
  - **Email Verification**: Verifikasi pendaftaran akun via tautan email atau kode JSON, lengkap dengan endpoint pengiriman ulang (_resend verification_).
  - **Email Change Verification**: Penggantian email akun dengan autentikasi password saat ini, verifikasi dikirim ke email tujuan baru, dan otomatis terverifikasi setelah tautan diklik.
- **User & Profile Management**: Pengelolaan data akun (ganti username, status verifikasi) dan profil personal (nama lengkap, bio, telepon, upload & hapus avatar dengan validasi _magic bytes_).
- **Password Hashing Standar Tertinggi**: Menggunakan **Argon2id** (RFC 9106 / rekomendasi OWASP) untuk keamanan kredensial maksimal.
- **Proteksi Brute-Force**: Dilengkapi rate limiter `10 request/menit per IP` pada endpoint registrasi, login, forgot password, dan reset password.
- **Background Periodic Worker**: Membersihkan refresh token dan auth token kedaluwarsa secara otomatis setiap 24 jam.
- **Modul CRUD Referensi**: Modul `posts` dan `categories` lengkap sebagai cetak biru (_blueprint_) pembuatan fitur CRUD di masa mendatang.
- **Developer Experience Prima**: Live reloading dengan **Air**, type-safe queries dengan **SQLC**, dan migrasi ter-embed via **golang-migrate**.

---

## 2. Tech Stack & Arsitektur

### Technology Stack

| Komponen               | Pilihan Teknologi                            | Keterangan                                                               |
| :--------------------- | :------------------------------------------- | :----------------------------------------------------------------------- |
| **Bahasa**             | Go (Golang) 1.25+                            | Standard library modern (`log/slog`, `embed`, `net/http`)                |
| **HTTP Router**        | Chi (`github.com/go-chi/chi/v5`)             | Ringan, cepat, idiomatik `net/http`, sub-router terstruktur              |
| **Identitas (ID)**     | ULID (`github.com/oklog/ulid/v2`)            | Crockford Base32 26-karakter (`CHAR(26)`)                                |
| **JWT**                | `github.com/golang-jwt/jwt/v5`               | Token stateless HS256 untuk otorisasi bearer                             |
| **Password Hashing**   | Argon2id (`github.com/alexedwards/argon2id`) | Standar keamanan kredensial modern terkini                               |
| **Rate Limiter**       | `github.com/go-chi/httprate`                 | Proteksi IP-based pada endpoint sensitif                                 |
| **Pluggable Mailer**   | Adapter Pattern (`internal/mailer`)          | `LogMailer` untuk dev lokal (terminal) & `ResendMailer` untuk production |
| **CORS**               | `github.com/go-chi/cors`                     | Cross-Origin Resource Sharing terkonfigurasi via `.env`                  |
| **Database**           | MySQL 8.0+                                   | InnoDB, collation `utf8mb4_0900_ai_ci`                                   |
| **SQL Generator**      | `sqlc` (`sqlc.yaml`)                         | Compile query SQL murni menjadi kode Go type-safe                        |
| **Database Migration** | `github.com/golang-migrate/migrate/v4`       | Migrasi DDL ter-embed ke dalam binary dengan `//go:embed`                |
| **Markdown Parser**    | Goldmark (`github.com/yuin/goldmark`)        | Parser Markdown ke HTML aman bebas XSS                                   |
| **Live Reload**        | Air (`.air.toml`)                            | Otomatis rebuild saat file source code berubah                           |

### Arsitektur Layering

```text
HTTP Request
     ↓
Chi Router & Global Middleware (Logger, Recoverer, CORS, RealIP)
     ↓
Route Group & Auth Middleware (JWTAuthMiddleware / RequireRole / RateLimiter)
     ↓
Handler Layer (internal/handlers) — Validasi DTO & format respon HTTP
     ↓
Service Layer (internal/service) — Logika bisnis & orkestrasi ──► Mailer Layer (Log / Resend)
     ↓
Repository Layer (internal/repository) — Interface & implementasi SQLC
     ↓
Database MySQL 8.0+
```

---

## 3. Struktur Direktori Proyek

```text
.
├── cmd/
│   └── server/
│       └── main.go                     # Composition Root: Bootstrap server, DI wiring, background worker
├── db/
│   ├── embed.go                        # Embed asset migrasi SQL ke binary Go
│   ├── migrations/                     # File migrasi DDL golang-migrate (UP/DOWN)
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   └── queries/                        # Query SQL mentah untuk sqlc generator
│       ├── auth_tokens.sql
│       ├── categories.sql
│       ├── post_views.sql
│       ├── posts.sql
│       ├── refresh_tokens.sql
│       ├── user_profiles.sql
│       └── users.sql
├── docs/
│   ├── ref/
│   │   └── architecture_reference.md   # Blueprint arsitektur backend Go
│   ├── learning_guide.md               # Panduan belajar arsitektur & konsep Go proyek ini
│   ├── starter_kit_specification.md    # Spesifikasi teknis & arsitektur Starter Kit
│   ├── technical_specification.md      # Spesifikasi teknis modul & skema database
│   └── postman_collection.json         # Koleksi Postman v2.1 siap pakai
├── internal/                           # Paket privat (hanya bisa diimpor modul ini)
│   ├── config/                         # Loader konfigurasi dari .env / OS env
│   ├── database/                       # MySQL pool, migrate runner, & idempotent seeder
│   ├── handlers/                       # HTTP Handlers (Auth, Users, Admin, Public)
│   ├── mailer/                         # Pluggable Mailer (LogMailer & ResendMailer)
│   ├── middleware/                     # JWT Auth, Role Guard, CORS middleware
│   ├── models/                         # Domain entities, input DTOs, & payload validators
│   ├── repository/                     # Repository interfaces & sqlc wrappers
│   │   └── sqlc/                       # Kode Go hasil generate otomatis sqlc
│   ├── routes/                         # Pendaftaran rute Chi & static file uploads
│   ├── service/                        # Business logic layer & interfaces
│   └── utils/                          # ULID, JWT, HashToken, JSON response helpers
├── .air.toml                           # Konfigurasi live-reload Air
├── .env.example                        # Template environment variables
├── api.http                            # Koleksi HTTP request REST Client
├── sqlc.yaml                           # Konfigurasi sqlc
└── README.md
```

---

## 4. Skema Database (MySQL 8.0+)

Semua entitas menggunakan tipe `CHAR(26)` untuk Primary Key dan Foreign Key:

1. **`users`**: Kredensial inti akun (`id`, `username`, `email`, `password_hash`, `role`, `status`, `email_verified_at`, `created_at`, `updated_at`).
2. **`user_profiles`**: Profil personal (`user_id` PK & FK, `full_name`, `bio`, `phone_number`, `avatar_url`, `created_at`, `updated_at`).
3. **`refresh_tokens`**: Stateful session tracker (`id`, `user_id`, `token_hash`, `token_family`, `is_revoked`, `expires_at`, `user_agent`, `ip_address`).
4. **`auth_tokens`**: Token sementara untuk verifikasi email, reset password, dan ubah email (`id`, `user_id`, `token_hash`, `token_type`, `payload`, `is_used`, `expires_at`, `created_at`).
5. **`categories`**: Modul referensi kategori konten (`id`, `name`, `slug`).
6. **`posts`**: Modul referensi konten artikel/blog (`id`, `type`, `title`, `slug`, `content`, `category_id`, `tags`, `status`, `published_at`).
7. **`post_views`**: Modul statistik views (`id`, `post_id`, `views`).

---

## 5. Rincian Endpoint API

### 5.1. Autentikasi (`/api/v1/auth`)

| Method         | Endpoint                           | Auth                  | Deskripsi                                                                                              |
| :------------- | :--------------------------------- | :-------------------- | :----------------------------------------------------------------------------------------------------- |
| `POST`         | `/api/v1/auth/register`            | Public (Rate Limited) | Registrasi pengguna baru (username, email, password, full_name). Otomatis mengirim email verifikasi.   |
| `POST`         | `/api/v1/auth/login`               | Public (Rate Limited) | Login pengguna/admin. Mengembalikan token hybrid (JSON & Cookie).                                      |
| `POST`         | `/api/v1/auth/refresh`             | Public / Cookie       | Memperoleh access token baru dengan mekanisme Refresh Token Rotation (RTR).                            |
| `POST`         | `/api/v1/auth/logout`              | Protected (Bearer)    | Mencabut refresh token dan menghapus cookie sesi.                                                      |
| `POST`         | `/api/v1/auth/change-password`     | Protected (Bearer)    | Mengganti password akun dan membatalkan seluruh sesi login lainnya.                                    |
| `POST`         | `/api/v1/auth/forgot-password`     | Public (Rate Limited) | Permintaan reset password via email. Anti-enumerasi (selalu 200 OK).                                   |
| `POST`         | `/api/v1/auth/reset-password`      | Public (Rate Limited) | Mengatur ulang password dengan token reset SHA-256 (TTL 15 mnt). Mencabut seluruh refresh token aktif. |
| `GET` / `POST` | `/api/v1/auth/verify-email`        | Public                | Verifikasi alamat email pengguna via query parameter (`?token=...`) atau JSON body.                    |
| `POST`         | `/api/v1/auth/resend-verification` | Public (Rate Limited) | Pengiriman ulang email verifikasi. Anti user-enumeration (selalu 200 OK).                              |

### 5.2. User Management (`/api/v1/users`)

| Method         | Endpoint                             | Auth               | Deskripsi                                                                                                    |
| :------------- | :----------------------------------- | :----------------- | :----------------------------------------------------------------------------------------------------------- |
| `GET`          | `/api/v1/users/me`                   | Protected (Bearer) | Mengambil informasi akun dan profil pengguna yang sedang login.                                              |
| `PUT`          | `/api/v1/users/profile`              | Protected (Bearer) | Memperbarui informasi profil (`full_name`, `bio`, `phone_number`).                                           |
| `POST`         | `/api/v1/users/avatar`               | Protected (Bearer) | Mengunggah foto profil avatar (multipart, JPG/PNG/WEBP maks 5MB).                                            |
| `DELETE`       | `/api/v1/users/avatar`               | Protected (Bearer) | Menghapus foto profil avatar.                                                                                |
| `PUT`          | `/api/v1/users/username`             | Protected (Bearer) | Mengubah username (validasi alfanumerik 3-30 karakter & anti-duplikasi).                                     |
| `GET`          | `/api/v1/users/status`               | Protected (Bearer) | Melihat status aktif akun dan status verifikasi email.                                                       |
| `POST`         | `/api/v1/users/email/request-change` | Protected (Bearer) | Mengajukan penggantian email akun dengan validasi password saat ini. Token dikirim ke email baru.            |
| `GET` / `POST` | `/api/v1/users/email/confirm-change` | Public             | Konfirmasi pergantian email via query parameter atau JSON body. Mengubah email dan menandai status verified. |

### 5.3. Admin & Content Management (`/api/v1/admin`)

_Hanya dapat diakses oleh akun dengan `role = 'admin'`._

| Method                   | Endpoint                        | Deskripsi                                                |
| :----------------------- | :------------------------------ | :------------------------------------------------------- |
| `GET` / `POST`           | `/api/v1/admin/posts`           | List seluruh post (draft & published) / Tambah post baru |
| `GET` / `PUT` / `DELETE` | `/api/v1/admin/posts/{id}`      | Detail / Perbarui / Hapus post berdasarkan ULID          |
| `GET` / `POST`           | `/api/v1/admin/categories`      | List seluruh kategori / Tambah kategori baru             |
| `PUT` / `DELETE`         | `/api/v1/admin/categories/{id}` | Perbarui / Hapus kategori berdasarkan ULID               |
| `POST`                   | `/api/v1/admin/upload`          | Upload file gambar umum ke direktori lokal               |

### 5.4. Public Content (`/api/v1/posts` & `/api/v1/categories`)

Endpoint publik read-only tanpa autentikasi:

- `GET /api/v1/posts` (Filter: `type`, `tag`, `featured`, `page`, `limit`)
- `GET /api/v1/posts/{slug}` (Detail post published & increment views)
- `GET /api/v1/categories` & `GET /api/v1/categories/{slug}/posts`
- URL Aliases: `/blog/{slug}`, `/articles/{slug}`, `/journal/{slug}`, `/categories/{slug}`

---

## 6. Panduan Menjalankan Proyek

### Persyaratan Sistem

- **Go 1.25+**
- **MySQL 8.0+**

### 1. Setup Environment (`.env`)

Salin file `.env.example` ke `.env`:

```powershell
Copy-Item .env.example .env
```

Sesuaikan konfigurasi koneksi MySQL dan kunci JWT pada file `.env`:

```ini
PORT=8080
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=go_crud_api

ADMIN_NAME=Administrator
ADMIN_EMAIL=admin@mmdbconnect.com
ADMIN_PASSWORD=adminpassword123

JWT_SECRET=super-secret-starter-kit-jwt-key-change-in-production
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Pluggable Mailer: 'log' (default lokal tanpa setup/akun) atau 'resend'
MAILER_DRIVER=log
RESEND_API_KEY=
MAIL_FROM=Go Starter Kit <onboarding@resend.dev>
FRONTEND_URL=http://localhost:3000
```

### 2. Inisialisasi Database Pertama Kali

Jalankan server dengan flag `-fresh` untuk membuat seluruh tabel dan menyemai akun administrator awal:

```powershell
go run cmd/server/main.go -fresh
```

### 3. Menjalankan Server Development (Hot-Reload)

Gunakan **Air** untuk proses development cepat:

```powershell
air
```

Atau gunakan perintah standar Go:

```powershell
go run cmd/server/main.go
```

### 4. Menjalankan Unit & Integration Test

```powershell
go test -v ./...
```

### 5. Menjalankan SQL Generator (SQLC)

Jika Anda menambah query SQL di `db/queries/*.sql`:

```powershell
sqlc generate
```

### 6. Panduan Menghubungkan Layanan Email Production (Resend)

Ketika Anda siap beralih dari pengujian lokal (`LogMailer`) ke pengiriman email sungguhan di staging/production:

1. **Daftar Akun Resend**:
   Kunjungi [resend.com](https://resend.com) dan buat akun (tier gratis mencakup 3.000 email/bulan dan 100 email/hari).

2. **Dapatkan API Key**:
   - Masuk ke dashboard Resend $\rightarrow$ menu **API Keys**.
   - Klik **Create API Key**, beri nama (misal: `go-starter-kit`), pilih permission `Full access`, lalu salin token yang diawali dengan `re_...`.

3. **Domain Pengirim**:
   - **Mode Testing Cepat**: Anda dapat langsung menggunakan sender bawaan Resend: `onboarding@resend.dev` (hanya bisa mengirim ke email yang Anda daftarkan di akun Resend).
   - **Mode Production**: Tambahkan domain Anda di menu **Domains**, masukkan DNS records (DKIM, SPF) ke registrar domain Anda, dan tunggu statusnya terverifikasi.

4. **Konfigurasi Environment (`.env` / `.env.docker`)**:
   Buka file `.env` dan perbarui variabel mailer:

   ```ini
   # Ubah driver dari 'log' ke 'resend'
   MAILER_DRIVER=resend

   # Masukkan API Key dari Resend
   RESEND_API_KEY=re_1234567890abcdef...

   # Sender email terverifikasi (atau onboarding@resend.dev untuk testing)
   MAIL_FROM=Go Starter Kit <onboarding@resend.dev>

   # URL frontend Anda tempat tautan email diarahkan
   FRONTEND_URL=https://app.domainanda.com
   ```

5. **Jalankan Ulang Aplikasi**:
   Mulai ulang server Go (`go run cmd/server/main.go` atau `docker compose up -d`). Seluruh alur verifikasi email pendaftaran, permintaan reset kata sandi, dan konfirmasi ubah email akan terkirim secara otomatis ke inbox pengguna via Resend REST API!

---

## 7. Pengujian API

- **Postman**: Impor file [postman_collection.json](docs/postman_collection.json). Koleksi ini sudah dilengkapi script otomatis untuk menyimpan `accessToken` dan `refreshToken` ke Postman Environment variable.
- **VS Code REST Client**: Buka file [api.http](api.http) untuk mengeksekusi request langsung dari editor Anda.
- **Spesifikasi Lengkap**: Rincian arsitektur lengkap dapat dibaca pada [starter_kit_specification.md](docs/starter_kit_specification.md).
