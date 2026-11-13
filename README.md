# mmdbconnect Personal Website — CMS (Blog, Article & Journal)

Sistem Content Management modular untuk mengelola dan mempublikasikan konten pada website pribadi **mmdbconnect** menggunakan **Go (Golang)** sebagai backend dan **MySQL 8.0+** sebagai database.

Sistem mendukung 3 jenis konten yang disimpan dalam satu tabel utama `posts`:

- **`blog`**: Tulisan personal, pengalaman, dan opini.
- **`article`**: Artikel teknis, tutorial, dan pembahasan mendalam.
- **`journal`**: Catatan pengembangan, eksperimen, dan dokumentasi.

---

## 1. Arsitektur & Teknologi

### Technology Stack

| Komponen                | Teknologi                                                          |
| ----------------------- | ------------------------------------------------------------------ |
| **Backend**             | Go (Golang) 1.25+                                                  |
| **HTTP Router**         | Chi (`github.com/go-chi/chi/v5`)                                   |
| **Rate Limiter**        | `github.com/go-chi/httprate` (proteksi login 10 req/menit per IP)  |
| **CORS**                | `github.com/go-chi/cors`                                           |
| **Database**            | MySQL 8.0+ (InnoDB, `utf8mb4_0900_ai_ci`)                          |
| **Database Driver**     | `go-sql-driver/mysql` via `database/sql`                           |
| **SQL Code Generation** | `sqlc`                                                             |
| **Database Migration**  | `golang-migrate` (`migrate/v4` dengan `//go:embed`)                |
| **Autentikasi**         | Secure HTTP-only Cookie Session (`SameSite=Lax`) + Bearer fallback |
| **Password Hashing**    | Argon2id (`github.com/alexedwards/argon2id`)                       |
| **Format Konten**       | Markdown (dengan safe HTML rendering via Goldmark)                 |
| **Penyimpanan Gambar**  | Local Filesystem (`/uploads`, validasi magic bytes)                |
| **Format API**          | RESTful / JSON                                                     |

### Backend Layering

```text
HTTP Handler
    ↓
Service
    ↓
Repository (sqlc queries + parameterized SQL)
    ↓
MySQL 8.0+
```

- **Handler (`internal/handlers`)**: HTTP request, response, cookie management, dan validasi input via service interface.
- **Service (`internal/service`)**: Logika bisnis (interface-driven), publication rules, slug generation, tags normalization, markdown rendering (`MarkdownRenderer`), dan SEO fallbacks.
- **Repository (`internal/repository`)**: Akses query database berbasis `sqlc` dan `database/sql`.
- **Database (`internal/database`)**: Connection pooling, database migrations via `golang-migrate`, dan data seeder independen.

---

## 2. Struktur Proyek

```text
.
├── cmd/
│   └── server/
│       └── main.go                  # Entry point & composition root
├── db/                              # Sumber tunggal aset SQL mentah
│   ├── embed.go                     # Ekspor file migrasi SQL ke Go runtime
│   ├── migrations/                  # Standar SQL migrations golang-migrate
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   └── queries/                     # SQL queries untuk sqlc generator
│       ├── categories.sql
│       ├── post_views.sql
│       ├── posts.sql
│       ├── sessions.sql
│       └── users.sql
├── docs/                            # Dokumentasi proyek & koleksi API
│   ├── ref/
│   │   └── architecture_reference.md # Blueprint arsitektur backend Go
│   ├── learning_guide.md            # Panduan belajar arsitektur & idiom Go proyek ini
│   ├── technical_specification.md   # Spesifikasi teknis, skema DB, & endpoint API
│   └── postman_collection.json      # Koleksi Postman v2.1 siap pakai
├── internal/
│   ├── config/                      # Konfigurasi aplikasi dari .env / OS env
│   ├── database/                    # Connection pool, migrate runner, & initial seeder
│   │   ├── database.go
│   │   ├── migrate.go
│   │   └── seeder.go
│   ├── handlers/                    # HTTP Handlers (Admin, Public, & handlers_test.go)
│   ├── middleware/                  # HTTP-only Cookie Auth & CORS middleware
│   ├── models/                      # Domain models, Request DTOs, & filter validators
│   ├── repository/                  # Repository interfaces & implementations
│   │   ├── sqlc/                    # Go code hasil generate otomatis dari sqlc
│   │   ├── category_repository.go
│   │   ├── post_repository.go
│   │   ├── session_repository.go
│   │   └── user_repository.go
│   ├── routes/                      # Pemetaan rute Chi, aliases, & rate limiter
│   ├── service/                     # Service interfaces, business logic, Argon2id, Goldmark
│   └── utils/                       # Slug generator, pagination, response helpers
├── .air.toml                        # Konfigurasi live-reload Air
├── .env.example                     # Contoh konfigurasi environment
├── api.http                         # Koleksi HTTP request REST Client
├── sqlc.yaml                        # Konfigurasi sqlc
└── README.md
```

---

## 3. Database Schema

### 3.1. `posts`

Tabel utama penyimpan seluruh konten:

- `id` (BIGINT UNSIGNED, PK, AUTO_INCREMENT)
- `type` (ENUM('blog','article','journal'), NOT NULL)
- `title` (VARCHAR(255), NOT NULL)
- `slug` (VARCHAR(255), UNIQUE, NOT NULL)
- `excerpt` (TEXT, NULLABLE)
- `content` (LONGTEXT, NOT NULL - Markdown)
- `cover_image` (VARCHAR(500), NULLABLE)
- `category_id` (BIGINT UNSIGNED, FK ke `categories.id` ON DELETE SET NULL)
- `tags` (JSON, NULLABLE - JSON Array)
- `meta_title` (VARCHAR(255), NULLABLE)
- `meta_description` (TEXT, NULLABLE)
- `is_featured` (BOOLEAN, DEFAULT FALSE)
- `status` (ENUM('draft','published'), DEFAULT 'draft')
- `published_at` (DATETIME, NULLABLE)
- `created_at` (DATETIME, NOT NULL)
- `updated_at` (DATETIME, NOT NULL)

**Constraints & Indexes:**

- `CHECK (status <> 'published' OR published_at IS NOT NULL)`
- `INDEX (type, status, published_at)`
- `INDEX (status, published_at)`
- `INDEX (category_id)`

### 3.2. `categories`

- `id` (BIGINT UNSIGNED, PK, AUTO_INCREMENT)
- `name` (VARCHAR(100), NOT NULL)
- `slug` (VARCHAR(120), UNIQUE, NOT NULL)
- `created_at` & `updated_at` (DATETIME, NOT NULL)

### 3.3. `post_views`

- `id` (BIGINT UNSIGNED, PK, AUTO_INCREMENT)
- `post_id` (BIGINT UNSIGNED, FK ke `posts.id` ON DELETE CASCADE, UNIQUE)
- `views` (BIGINT UNSIGNED, DEFAULT 0)
- `updated_at` (DATETIME, NOT NULL)

### 3.4. `users` & `sessions`

- `users`: Administrator tunggal dengan password hash **Argon2id**.
- `sessions`: Token sesi aman 32-byte untuk otentikasi **HTTP-only Cookie Session**.

---

## 4. Aturan Bisnis & Logika Publikasi

1. **Publication Status**:
   - Konten baru berstatus default `draft`.
   - Saat pertama kali dipublikasikan (`status: published`), `published_at` diisi waktu saat ini (UTC).
   - Saat konten diperbarui, `updated_at` berubah tanpa mengubah nilai `published_at`.
   - Jika konten dikembalikan ke `draft`, nilai `published_at` **dipertahankan**, namun konten disembunyikan dari Public API (mengembalikan 404).
2. **Slug Unik**:
   - Dihasilkan otomatis dari `title` menjadi huruf kecil dan tanda hubung (misal: `building-rest-apis-with-go`).
   - Dapat diubah manual oleh administrator dan divalidasi keunikannya secara global di database.
3. **Tags (JSON Array)**:
   - Disimpan langsung sebagai JSON Array di kolom `posts.tags`.
   - Dinormalisasi (spasi dibersihkan dan diduplikasi secara case-insensitive).
   - Difilter pada query publik menggunakan fungsi MySQL `JSON_CONTAINS()`.
4. **SEO Metadata Fallback**:
   - Jika `meta_title` kosong, otomatis menggunakan `title`.
   - Jika `meta_description` kosong, otomatis menggunakan `excerpt`.
5. **Safe Markdown Rendering**:
   - Konten Markdown dirender ke format HTML yang aman via **Goldmark** tanpa mengizinkan script injeksi berbahaya (XSS Protection).
6. **Statistik Views**:
   - Saat detail artikel diakses publik (`GET /api/v1/posts/{slug}`), counter kunjungan pada tabel `post_views` bertambah secara otomatis.

---

## 5. Daftar Endpoint API

### 5.1. Endpoint Publik

| Method | Endpoint                          | Deskripsi                     | Query Params                               |
| ------ | --------------------------------- | ----------------------------- | ------------------------------------------ |
| `GET`  | `/api/v1/posts`                   | Daftar konten published       | `type`, `tag`, `featured`, `page`, `limit` |
| `GET`  | `/api/v1/posts/{slug}`            | Detail konten published       | -                                          |
| `GET`  | `/api/v1/categories`              | Daftar seluruh kategori       | -                                          |
| `GET`  | `/api/v1/categories/{slug}/posts` | Konten published per kategori | `page`, `limit`, `type`, `tag`, `featured` |
| `GET`  | `/blog/{slug}`                    | Alias URL publik Blog         | -                                          |
| `GET`  | `/articles/{slug}`                | Alias URL publik Article      | -                                          |
| `GET`  | `/journal/{slug}`                 | Alias URL publik Journal      | -                                          |
| `GET`  | `/categories/{slug}`              | Alias URL publik Kategori     | -                                          |
| `GET`  | `/uploads/{file}`                 | Penyajian file cover image    | -                                          |

### 5.2. Endpoint Administrator

| Method   | Endpoint                        | Deskripsi                              | Proteksi                                |
| -------- | ------------------------------- | -------------------------------------- | --------------------------------------- |
| `POST`   | `/api/v1/admin/login`           | Login admin (Argon2id) & Set Cookie    | Publik (Rate Limit 10 req/menit per IP) |
| `POST`   | `/api/v1/admin/logout`          | Logout admin & Clear Cookie            | Auth Cookie                             |
| `GET`    | `/api/v1/admin/me`              | Profil admin yang sedang login         | Auth Cookie                             |
| `GET`    | `/api/v1/admin/posts`           | Semua konten (Draft & Published)       | Auth Cookie                             |
| `POST`   | `/api/v1/admin/posts`           | Membuat konten baru                    | Auth Cookie                             |
| `GET`    | `/api/v1/admin/posts/{id}`      | Detail konten berdasarkan ID           | Auth Cookie                             |
| `PUT`    | `/api/v1/admin/posts/{id}`      | Memperbarui konten                     | Auth Cookie                             |
| `DELETE` | `/api/v1/admin/posts/{id}`      | Menghapus konten                       | Auth Cookie                             |
| `GET`    | `/api/v1/admin/categories`      | Daftar semua kategori                  | Auth Cookie                             |
| `POST`   | `/api/v1/admin/categories`      | Membuat kategori baru                  | Auth Cookie                             |
| `PUT`    | `/api/v1/admin/categories/{id}` | Memperbarui kategori                   | Auth Cookie                             |
| `DELETE` | `/api/v1/admin/categories/{id}` | Menghapus kategori                     | Auth Cookie                             |
| `POST`   | `/api/v1/admin/upload`          | Upload cover image ke filesystem lokal | Auth Cookie                             |

---

## 6. Panduan Menjalankan

### Persyaratan Sistem

- **Go 1.25+**
- **MySQL 8.0+**

### 1. Konfigurasi Lingkungan (`.env`)

Salin file `.env.example` menjadi `.env`:

```bash
cp .env.example .env
```

Sesuaikan parameter database di file `.env`:

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

COOKIE_SECURE=false
COOKIE_DOMAIN=
UPLOAD_DIR=./uploads
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 2. Menjalankan Server

Jalankan aplikasi secara langsung:

```bash
go run cmd/server/main.go
```

Jika ingin mereset dan memigrasikan database dari nol:

```bash
go run cmd/server/main.go -fresh
```

Aplikasi secara otomatis:

1. Membuka koneksi pool MySQL.
2. Mengeksekusi database migrations via `golang-migrate`.
3. Mengisi data awal administrator, kategori, dan sample post jika tabel masih kosong.
4. Menjalankan server HTTP di port 8080 dengan mekanisme _Graceful Shutdown_.

### 3. Menjalankan Unit & Integration Test

```bash
go test -v ./...
```

### 4. Menjalankan Code Generator (sqlc)

Jika mengubah query di `db/queries/*.sql` atau skema migrasi di `db/migrations/*.sql`:

```bash
sqlc generate
```

### 5. Pengujian API via REST Client & Postman

- Gunakan file [`api.http`](./api.http) di VS Code / Cursor / JetBrains untuk menguji seluruh endpoint secara langsung.
- Atau impor [`docs/postman_collection.json`](./docs/postman_collection.json) ke Postman / Insomnia.

### 6. Referensi Arsitektur Proyek

Lihat [`docs/ref/architecture_reference.md`](./docs/ref/architecture_reference.md) untuk blueprint arsitektur lengkap beserta struktur folder standar yang digunakan proyek ini.
