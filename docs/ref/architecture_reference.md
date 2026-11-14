# Referensi Arsitektur Backend Go (Starter Kit Blueprint)

Dokumen ini adalah **cetak biru (*blueprint*) arsitektur resmi** dari **Go REST API Starter Kit (`go-crud-api`)**. Gunakan dokumen ini sebagai acuan standar setiap kali memulai proyek backend Go baru agar struktur folder, pemisahan layer, identitas berbasis ULID, autentikasi modern (JWT + RTR), konvensi kode, keamanan, dan alur pengembangannya konsisten dan teruji.

---

## 1. Tech Stack & Dependensi Inti

| Komponen | Pilihan Standar | Catatan Penggunaan |
| :--- | :--- | :--- |
| **Bahasa** | Go `1.25+` | Menggunakan pustaka standar modern (`log/slog`, `embed`, `net/http`) |
| **HTTP Router** | `github.com/go-chi/chi/v5` | Ringan, idiomatik `net/http`, kompatibel penuh dengan `http.Handler` |
| **Identitas (ID)** | `github.com/oklog/ulid/v2` | ULID 26-karakter (`CHAR(26)`), aman dari URL enumeration, terurut kronologis |
| **Autentikasi** | `github.com/golang-jwt/jwt/v5` | Access Token stateless HS256 + Refresh Token Rotation (RTR) |
| **Password Hashing** | `github.com/alexedwards/argon2id` | Standar hashing kredensial modern (RFC 9106) |
| **Rate Limiter** | `github.com/go-chi/httprate` | Proteksi endpoint sensitif (login/register) berbasis IP (10 req/menit) |
| **CORS** | `github.com/go-chi/cors` | Konfigurasi origin via environment variable |
| **Database Driver** | `database/sql` + `go-sql-driver/mysql` | Kontrol penuh atas connection pool & transaksi |
| **SQL Codegen** | `sqlc` (`sqlc.yaml`) | Type-safe Go code dari query SQL murni tanpa overhead ORM |
| **Migrasi DB** | `github.com/golang-migrate/migrate/v4` | File SQL di-embed ke binary via `//go:embed` dengan dukungan flag `-fresh` |
| **Live Reload** | Air (`.air.toml`) | Auto-rebuild saat development |

---

## 2. Struktur Folder Standar

Setiap proyek backend baru mengikuti struktur direktori berikut:

```text
<nama-project>/
├── cmd/
│   └── server/
│       └── main.go                     # Composition root: bootstrap, wiring DI, background worker, graceful shutdown
├── db/
│   ├── embed.go                        # Export migrations FS via //go:embed migrations/*.sql
│   ├── migrations/                     # File migrasi SQL berurutan (000001_init_schema.up.sql & .down.sql)
│   └── queries/                        # File .sql mentah sebagai sumber generator sqlc
├── docs/
│   ├── ref/
│   │   └── architecture_reference.md   # Blueprint arsitektur ini
│   ├── starter_kit_specification.md    # Spesifikasi teknis & arsitektur proyek
│   ├── learning_guide.md               # Panduan belajar arsitektur proyek
│   └── postman_collection.json         # Koleksi endpoint siap uji untuk Postman/Insomnia
├── internal/
│   ├── config/
│   │   └── config.go                   # Struct Config & loader env (dengan nilai fallback aman)
│   ├── database/
│   │   ├── database.go                 # Init koneksi DB, pool settings, ping, panggil migrate & seeder
│   │   ├── migrate.go                  # Runner golang-migrate (+ dukungan flag -fresh untuk reset dev)
│   │   └── seeder.go                   # Idempotent initial data seeder dengan ULID
│   ├── handlers/
│   │   ├── auth_handler.go             # Handler login, register, refresh (RTR), logout, change-password
│   │   ├── user_handler.go             # Handler profil, avatar upload/delete, username, status akun
│   │   ├── admin_*_handler.go          # Handler endpoint terproteksi khusus role admin
│   │   ├── public_*_handler.go         # Handler endpoint publik
│   │   └── handlers_test.go            # Integration-style HTTP tests via httptest + in-memory repo
│   ├── middleware/
│   │   ├── auth.go                     # JWTAuthMiddleware, RequireRole, & helper context
│   │   └── cors.go                     # Konfigurasi middleware CORS
│   ├── models/
│   │   ├── auth.go                     # DTO autentikasi, TokenResponse, RegisterInput, RefreshToken
│   │   ├── user.go                     # Entitas User, UserProfile, UserWithProfile, DTOs
│   │   └── <entity>.go                 # Domain struct, Input DTO, & metode Validate()
│   ├── repository/
│   │   ├── sqlc/                       # Hasil generate otomatis dari `sqlc generate` (jangan edit manual)
│   │   └── <entity>_repository.go      # Interface repository, sentinel error, wrapper sqlc & dynamic query
│   ├── routes/
│   │   └── routes.go                   # Registrasi global middleware, health check, static files, & sub-routes
│   ├── service/
│   │   ├── auth_service.go             # Logika autentikasi, RTR, Reuse Detection, Argon2id
│   │   ├── user_service.go             # Logika pengelolaan profil, avatar, & username
│   │   ├── <domain>_service.go         # Domain service lainnya (posts, categories, markdown)
│   │   └── *_test.go                   # Unit test business logic dengan mock repository
│   └── utils/
│       ├── ulid.go                     # Generator & validator ULID 26-karakter
│       ├── jwt.go                      # Generate/Validate JWT access token & SHA-256 token hashing
│       ├── pagination.go               # Paginasi helper
│       └── response.go                 # Helper JSON() & Error() standar
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
[ routes + middleware ]     ──► JWT Bearer Auth / Role Guard / Rate Limiter
     │
     ▼
[ internal/handlers ]       ──► Validasi DTO input, mapping HTTP status, hybrid delivery (Cookie + JSON)
     │ (via Service Interface)
     ▼
[ internal/service ]        ──► Business rules, password hash Argon2id, token family, reuse detection
     │ (via Repository Interface)
     ▼
[ internal/repository ]     ──► Penugasan ULID sebelum INSERT, wrapper query sqlc, mapping error DB
     │
     ▼
[ internal/repository/sqlc ] ──► Kode SQL type-safe hasil compile sqlc
     │
     ▼
  Database (MySQL 8.0+)
```

---

## 4. Standar Autentikasi & Keamanan Bawaan

1. **Identitas Global ULID**:
   - Seluruh primary key dan foreign key menggunakan `CHAR(26)`.
   - Di-generate di sisi aplikasi Go (`utils.NewULID()`) sebelum query dijalankan.
2. **Access Token Stateless (JWT)**:
   - Masa berlaku pendek (15 menit).
   - Memuat claims penting: `sub` (ULID), `user_id`, `username`, `email`, `role`, `exp`, `iat`.
3. **Refresh Token Stateful & Rotation (RTR)**:
   - Token mentah berupa string acak kriptografis 64-karakter.
   - Disimpan di database hanya dalam bentuk **SHA-256 Hash** (`token_hash`).
   - Setiap kali refresh token digunakan, token lama dicabut (`is_revoked = true`) dan token baru diterbitkan di bawah `token_family` yang sama.
   - **Reuse Detection**: Jika token yang sudah di-revoke dicoba digunakan kembali, sistem mencabut seluruh keluarga token (`token_family`) sebagai mitigasi pembajakan sesi.
4. **Delivery Hybrid**:
   - Response JSON body: `{ access_token, refresh_token, token_type, expires_in }`.
   - Response Cookie: `HttpOnly = true`, `SameSite = Lax`, `Path = /api/v1/auth`.
5. **Password Hashing Argon2id**:
   - Selalu menggunakan `argon2id.CreateHash` dengan default OWASP parameter.
6. **Upload File Aman**:
   - Validasi format dan **magic bytes** konten aktual via `http.DetectContentType`.
   - Menggunakan stream rewind (`io.SeekStart`) sebelum menyalin konten.

---

## 5. Strategi Verifikasi & Testing

Sebelum melakukan commit, pastikan seluruh pengujian lolos:

```powershell
go build ./...
go test -v ./...
go vet ./...
```
