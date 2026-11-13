# Referensi Arsitektur Backend Go (Blueprint Proyek)

Dokumen ini adalah **cetak biru (blueprint) arsitektur resmi** yang diekstrak dari proyek `go-crud-api`. Gunakan dokumen ini sebagai acuan standar setiap kali memulai proyek backend Go baru agar struktur folder, pemisahan layer, konvensi kode, keamanan, dan alur pengembangannya konsisten.

---

## 1. Tech Stack & Dependensi Inti

| Komponen             | Pilihan Standar                                            | Catatan Penggunaan                                                   |
| -------------------- | ---------------------------------------------------------- | -------------------------------------------------------------------- |
| **Bahasa**           | Go `1.25+`                                                 | Memanfaatkan stdlib modern (`log/slog`, `embed`, `net/http`)         |
| **HTTP Router**      | `github.com/go-chi/chi/v5`                                 | Ringan, idiomatik `net/http`, kompatibel penuh dengan `http.Handler` |
| **CORS**             | `github.com/go-chi/cors`                                   | Konfigurasi origin via environment variable                          |
| **Rate Limiter**     | `github.com/go-chi/httprate`                               | Proteksi endpoint sensitif (mis. login) berbasis key/IP              |
| **Database Driver**  | `database/sql` + driver DB (contoh: `go-sql-driver/mysql`) | Kontrol penuh atas connection pool & query                           |
| **SQL Codegen**      | `sqlc` (`sqlc.yaml`)                                       | Type-safe Go code dari query SQL murni                               |
| **Migrasi DB**       | `github.com/golang-migrate/migrate/v4`                     | File SQL di-embed ke binary via `//go:embed`                         |
| **Password Hashing** | `github.com/alexedwards/argon2id`                          | Standar hashing modern pengganti bcrypt                              |
| **Env Config**       | `github.com/joho/godotenv`                                 | Load `.env` untuk lokal, fallback ke OS env di production            |
| **Live Reload**      | Air (`.air.toml`)                                          | Auto-rebuild saat development                                        |

---

## 2. Struktur Folder Standar

Setiap proyek backend baru mengikuti struktur direktori berikut:

```text
<nama-project>/
├── cmd/
│   └── server/
│       └── main.go                     # Composition root: bootstrap, wiring DI, background job, graceful shutdown
├── db/
│   ├── embed.go                        # Export migrations FS via //go:embed migrations/*.sql
│   ├── migrations/                     # File migrasi SQL berurutan (000001_init_schema.up.sql & .down.sql)
│   └── queries/                        # File .sql mentah sebagai sumber generator sqlc
├── docs/
│   ├── ref/
│   │   └── architecture_reference.md   # Blueprint arsitektur ini
│   ├── technical_specification.md      # Spesifikasi fitur, ERD, & kontrak API proyek
│   └── postman_collection.json         # Koleksi endpoint siap uji untuk Postman/Insomnia
├── internal/
│   ├── config/
│   │   └── config.go                   # Struct Config & loader env (dengan nilai fallback aman)
│   ├── database/
│   │   ├── database.go                 # Init koneksi DB, pool settings, ping, panggil migrate & seeder
│   │   ├── migrate.go                  # Runner golang-migrate (+ dukungan flag -fresh untuk reset dev)
│   │   └── seeder.go                   # Idempotent initial data seeder (tanpa import internal/service)
│   ├── handlers/
│   │   ├── admin_*_handler.go          # Handler endpoint terproteksi (admin/private)
│   │   ├── public_*_handler.go         # Handler endpoint publik
│   │   └── handlers_test.go            # Integration-style HTTP tests via httptest + in-memory repo
│   ├── middleware/
│   │   ├── auth.go                     # Validasi session/token + helper context (GetUserFromContext, dll.)
│   │   └── cors.go                     # Konfigurasi middleware CORS
│   ├── models/
│   │   └── <entity>.go                 # Domain struct, Input DTO, metode Validate(), & filter validator
│   ├── repository/
│   │   ├── sqlc/                       # Hasil generate otomatis dari `sqlc generate` (jangan edit manual)
│   │   └── <entity>_repository.go      # Interface repository, sentinel error, wrapper sqlc & dynamic query
│   ├── routes/
│   │   └── routes.go                   # Registrasi global middleware, health check, static files, & routing API
│   ├── service/
│   │   ├── <domain>_service.go         # Interface service, implementasi business logic, & sentinel error
│   │   └── *_test.go                   # Unit test business logic dengan mock repository/dependency
│   └── utils/
│       ├── pagination.go               # ParsePagination, BuildPagination, CalculateOffset
│       ├── response.go                 # Helper JSON() & Error() standar
│       └── <helper>.go                 # Utility murni (mis. slug generator) + unit test
├── .air.toml                           # Konfigurasi live-reload Air
├── .env.example                        # Template environment variables
├── .gitignore
├── api.http                            # Request testing cepat untuk VS Code REST Client
├── go.mod                              # Module path standar: github.com/<username>/<repo>
├── go.sum
├── README.md
└── sqlc.yaml                           # Konfigurasi generator sqlc
```

---

## 3. Arsitektur Berlapis & Aturan Dependensi

Proyek menggunakan **Layered Architecture** dengan **Dependency Inversion Principle (DIP)**. Setiap layer atas hanya bergantung pada **interface** yang disediakan atau dibutuhkan, bukan concrete struct.

### Diagram Alur Request & Dependensi

```text
HTTP Request
     │
     ▼
[ routes + middleware ]     ──► Bergantung pada interface (mis. AuthServiceInterface)
     │
     ▼
[ internal/handlers ]       ──► Validasi HTTP input, mapping error ke status code, render JSON
     │ (via Service Interface)
     ▼
[ internal/service ]        ──► Aturan bisnis, orkestrasi, normalisasi, hashing, integrasi helper
     │ (via Repository & Helper Interface)
     ▼
[ internal/repository ]     ──► Implementasi akses data (sqlc + dynamic SQL), mapping error DB
     │
     ▼
[ internal/repository/sqlc ] ──► Kode SQL type-safe hasil generate
     │
     ▼
  Database (MySQL)
```

### Aturan Batas Antar Package (Strict Boundaries)

1. **`cmd/server/main.go`** adalah satu-satunya **Composition Root** yang merangkai concrete implementation (`New*Repository` → `New*Service` → `New*Handler` → `routes.Setup`). Tidak menggunakan framework DI otomatis agar wiring selalu eksplisit dan mudah ditelusuri.
2. **`internal/handlers`** tidak boleh mengimpor `internal/repository` atau `database/sql`. Handler hanya memanggil interface `service.*Interface`, membaca/menulis DTO dari `internal/models`, dan merender response lewat `internal/utils`.
3. **`internal/service`** tidak boleh mengimpor `net/http` atau `internal/handlers`. Service bekerja dengan `context.Context`, `internal/models`, interface dari `internal/repository`, serta interface internal sendiri (contoh: `MarkdownRenderer`).
4. **`internal/repository`** adalah satu-satunya layer yang mengerti `database/sql`, `sqlc`, dan error spesifik driver DB (mis. MySQL error `1062` → `repository.ErrDuplicate`, `sql.ErrNoRows` → `repository.ErrNotFound`).
5. **`internal/database`** tidak boleh mengimpor `internal/service` (agar tidak ada coupling bolak-balik). Jika seeder perlu hashing password, gunakan library crypto (`argon2id`) langsung.
6. **`internal/models`** dan **`internal/utils`** harus ringan dan bebas dari dependensi ke `handlers`, `service`, maupun `repository`.

---

## 4. Konvensi & Pola Desain Per Layer

### 4.1. Composition Root (`cmd/server/main.go`)

Urutan bootstrap standar di `main()`:

1. **Structured Logging**: inisialisasi `slog.NewJSONHandler(os.Stdout, ...)` sebagai default logger (`slog.SetDefault`).
2. **CLI Flags**: sediakan flag development seperlunya (mis. `-fresh` untuk reset tabel sebelum migrasi ulang).
3. **Config Loader**: muat konfigurasi melalui `config.Load()`.
4. **Database Init**: buka koneksi, set pool (`MaxIdleConns`, `MaxOpenConns`, `ConnMaxLifetime`), ping, jalankan migrasi, lalu jalankan seeder.
5. **Wiring DI Manual**: inisialisasi berurutan `Repositories` → `Services` → `Handlers` → `routes.Setup(...)`.
6. **Background Worker**: jalankan goroutine periodik (mis. cleanup session kedaluwarsa tiap 1 jam) dengan `context.WithCancel` + `defer stopCleanup()`.
7. **HTTP Server & Graceful Shutdown**:
   - Set `ReadTimeout`, `WriteTimeout`, dan `IdleTimeout` pada `http.Server`.
   - Dengarkan `os.Interrupt` dan `syscall.SIGTERM`.
   - Panggil `srv.Shutdown(ctx)` dengan timeout (mis. 10 detik) sebelum keluar.

### 4.2. Database, Migrasi, & `sqlc`

- **Migrasi (`db/migrations/` & `db/embed.go`)**:
  - Selalu sediakan pasangan `.up.sql` dan `.down.sql`.
  - Gunakan `//go:embed migrations/*.sql` di `db/embed.go` agar binary production self-contained tanpa bergantung pada file eksternal.
  - Terapkan integritas data langsung di skema DB (`FOREIGN KEY ... ON DELETE`, `UNIQUE KEY`, `CHECK` constraint, `ENUM`, dan composite `INDEX` sesuai pola query utama).
- **Repository (`internal/repository/`)**:
  - Query CRUD dan lookup statis ditulis di `db/queries/*.sql` lalu di-generate ke `internal/repository/sqlc/` memakai `sqlc`.
  - Query dengan filter dinamis + paginasi (`ListPublic`, `ListAdmin`) dapat menggunakan `r.sqlDB.QueryContext` dengan parameterized placeholder (`?`) dan helper `scan*` / `mapRowTo*` agar tidak terjadi duplikasi mapping kolom.
  - Selalu definisikan **repository interface** dan **sentinel error** (`ErrNotFound`, `ErrDuplicate`) di package `repository` agar layer `service` tidak pernah bergantung pada `sql.ErrNoRows` atau driver MySQL secara langsung.

### 4.3. Models & Validasi (`internal/models/`)

- Pisahkan **Domain Entity** (mis. `Post`, `User`, `Category`, `Session`) dari **Request DTO** (mis. `CreatePostInput`, `UpdatePostInput`, `LoginInput`, `CategoryInput`).
- Gunakan pointer (`*string`, `*uint64`, `*time.Time`) untuk field nullable agar pemetaan ke JSON `null` dan `sql.Null*` eksplisit.
- Sembunyikan field sensitif dengan tag `json:"-"` (contoh: `User.PasswordHash`) atau sediakan struct projection aman (contoh: `UserProfile`).
- Taruh validasi input dasar pada method `(i *Input) Validate() error` dan validasi enum/filter berulang pada fungsi helper di `models` (contoh: `ValidateType`, `ValidateStatus`) agar tidak terjadi duplikasi validasi di banyak handler.

### 4.4. Service Layer (`internal/service/`)

- Setiap service mendefinisikan **interface kontrak** (`AuthServiceInterface`, `PostServiceInterface`, `CategoryServiceInterface`) sehingga handler dan router hanya bergantung pada interface tersebut.
- Service memiliki **domain sentinel errors** sendiri (contoh: `ErrPostNotFound`, `ErrPostSlugExists`, `ErrInvalidCredentials`) yang diterjemahkan dari error repository.
- Hindari query DB berulang untuk data yang sudah dimiliki (contoh: pada `GetPublishedBySlug`, lakukan `GetPublishedBySlug` + `IncrementViews` lalu increment counter di struct lokal ketimbang query ulang ke DB).
- Abstraksikan library pihak ketiga di balik interface kecil bila diperlukan di business logic (contoh: `MarkdownRenderer` di `markdown_service.go`).

### 4.5. Handler & Error Handling (`internal/handlers/`, `internal/utils/`)

- Gunakan helper standar dari `internal/utils/response.go`:
  - `utils.JSON(w, status, data)`
  - `utils.Error(w, status, message)` → format konsisten `{"error": "..."}`.
- **Aturan Keamanan Error**:
  - `400 Bad Request`: validasi input gagal atau payload JSON tidak valid.
  - `401 Unauthorized`: sesi/token tidak ada, tidak valid, atau kedaluwarsa.
  - `404 Not Found` / `409 Conflict`: mapping eksplisit dari sentinel error service (`errors.Is`).
  - `500 Internal Server Error`: **jangan pernah** menggabungkan `err.Error()` ke response client. Log detail error dengan `slog.Error(...)` dan kirim pesan generik ke client.
- Gunakan `utils.ParsePagination`, `utils.BuildPagination`, dan `utils.CalculateOffset` untuk setiap endpoint list yang membutuhkan paginasi.

### 4.6. Routing & Middleware (`internal/routes/`, `internal/middleware/`)

Urutan middleware global di `routes.Setup`:

1. `chimiddleware.RequestID`
2. `chimiddleware.RealIP` _(catatan: jika server terekspos langsung tanpa reverse proxy terpercaya, pertimbangkan `ClientIPFromRemoteAddr` atau `ClientIPFromXFF` dengan CIDR proxy)_
3. `chimiddleware.Logger`
4. `chimiddleware.Recoverer`
5. `middleware.CORS(cfg.AllowedOrigins)`

Konvensi endpoint:

- Health check di root: `GET /health` → `{"status":"ok"}`.
- Versioning API di bawah prefix: `/api/v1/...`.
- Pisahkan grup publik dan terproteksi (`admin.Group` dengan `protected.Use(middleware.AuthMiddleware(authSvc))`).
- Pasang rate limiter (`httprate.LimitBy`) pada endpoint rawan brute-force seperti `POST /api/v1/admin/login`.

---

## 5. Standar Keamanan Bawaan

Setiap proyek baru berbasis blueprint ini mempertahankan praktik keamanan berikut:

1. **Password Hashing**: Argon2id (`argon2id.DefaultParams`), bukan MD5/SHA/plain bcrypt.
2. **Session Management**:
   - Session ID 32-byte acak dari `crypto/rand` (hex-encoded 64 karakter).
   - Disimpan di **HTTP-only Cookie** (`SameSite=Lax`, `Secure` dikontrol env `COOKIE_SECURE`) dengan dukungan fallback header `Authorization: Bearer <token>` untuk REST client.
   - Validasi ganda (defense-in-depth): filter `expires_at > NOW` di SQL + cek `session.IsExpired()` di service untuk mengantisipasi clock drift.
   - Pembersihan sesi kedaluwarsa secara berkala di background goroutine.
3. **Upload File Aman**:
   - Batasi ukuran form (`ParseMultipartForm(10 << 20)`).
   - Validasi **ekstensi file** dan **magic bytes** konten aktual (`http.DetectContentType`) untuk mencegah spoofing file berbahaya.
   - Rewind pointer file (`io.SeekStart`) setelah membaca header bytes.
   - Generate nama file unik acak (`crypto/rand`) + slug bersih.
4. **Anti-XSS Rendering**: Renderer Markdown (`goldmark`) tanpa `html.WithUnsafe()` agar HTML mentah berbahaya tidak lolos.
5. **SQL Injection Prevention**: Seluruh query menggunakan `sqlc` atau placeholder terparameter (`?`); tidak ada string concatenation dari input user ke dalam klausa SQL.

---

## 6. Strategi Testing

Gunakan pendekatan piramida testing ringan yang cepat dijalankan tanpa ketergantungan database nyata untuk unit/handler test:

- **Utils Test (`internal/utils/*_test.go`)**: Table-driven unit test untuk fungsi murni (slug generator, pagination).
- **Service Test (`internal/service/*_test.go`)**: Uji business rules, edge cases (slug collision, validasi relasi, fallback SEO, single-query behavior) dengan **in-memory mock repository** & **mock interface** (`MarkdownRenderer`).
- **Handler / Router Test (`internal/handlers/handlers_test.go`)**: Bangun router penuh via `routes.Setup(...)` dengan in-memory repository, lalu uji alur HTTP end-to-end memakai `net/http/httptest` (status code, cookie `HttpOnly`, auth guard `401`, rate limit `429`, MIME validation upload).

Perintah verifikasi standar sebelum commit:

```bash
go build ./...
go test -v ./...
go vet ./...
```

---

## 7. Checklist Memulai Proyek Backend Baru

Saat membuat proyek baru dengan blueprint ini, ikuti urutan berikut:

1. **Inisialisasi Modul & Struktur**
   - `go mod init github.com/<username>/<nama-project>`
   - Buat kerangka folder: `cmd/server`, `db/migrations`, `db/queries`, `docs/ref`, `internal/{config,database,handlers,middleware,models,repository,routes,service,utils}`.
   - Salin `.air.toml`, `.env.example`, `.gitignore`, dan `sqlc.yaml`.
2. **Siapkan Fondasi**
   - Salin `internal/config/config.go`, `internal/utils/response.go`, `internal/utils/pagination.go`, dan `internal/middleware/cors.go`.
3. **Desain Skema & Query**
   - Tulis `db/migrations/000001_init_schema.up.sql` & `.down.sql` serta `db/embed.go`.
   - Tulis query dasar di `db/queries/*.sql` lalu jalankan `sqlc generate`.
4. **Bangun Layer dari Dalam ke Luar**
   - `internal/models/` → struct domain, DTO, dan `Validate()`.
   - `internal/repository/` → interface, sentinel errors, implementasi wrapper `sqlc`.
   - `internal/service/` → interface service, sentinel errors, dan business logic.
   - `internal/handlers/` → handler HTTP tipis yang memanggil service interface + log error via `slog.Error`.
   - `internal/routes/routes.go` → pasang middleware global, rate limiter, dan grup route.
5. **Rangkai di Composition Root**
   - Lengkapi `cmd/server/main.go` (logger JSON, config, DB init, DI wiring, background workers, graceful shutdown).
6. **Lengkapi Test & Dokumentasi**
   - Tulis test di `internal/utils`, `internal/service`, dan `internal/handlers`.
   - Buat `api.http` dan `docs/postman_collection.json` sebagai dokumentasi uji endpoint.
