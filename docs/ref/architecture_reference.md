# Referensi Arsitektur Backend Go (Starter Kit Blueprint)

Dokumen ini adalah **cetak biru (_blueprint_) arsitektur resmi** dari **Go REST API Starter Kit (`go-crud-api`)**. Gunakan dokumen ini sebagai acuan standar setiap kali memulai proyek backend Go baru agar struktur folder, pemisahan layer, identitas berbasis ULID, autentikasi modern (JWT + RTR), konvensi kode, keamanan, dan alur pengembangannya konsisten dan teruji.

---

## 1. Tech Stack & Dependensi Inti

| Komponen                   | Pilihan Standar                        | Catatan Penggunaan                                                                 |
| :------------------------- | :------------------------------------- | :--------------------------------------------------------------------------------- |
| **Bahasa**                 | Go `1.25+`                             | Menggunakan pustaka standar modern (`log/slog`, `embed`, `net/http`)               |
| **HTTP Router**            | `github.com/go-chi/chi/v5`             | Ringan, idiomatik `net/http`, kompatibel penuh dengan `http.Handler`               |
| **Identitas (ID)**         | `github.com/oklog/ulid/v2`             | ULID 26-karakter (`CHAR(26)`), aman dari URL enumeration, terurut kronologis       |
| **Autentikasi**            | `github.com/golang-jwt/jwt/v5`         | Access Token stateless HS256 + Refresh Token Rotation (RTR)                        |
| **Otorisasi (RBAC)**       | RBAC Middleware + Wildcard Bypass      | Multi-role, pola `resource.action`, Super Admin wildcard (`*`), safeguard proteksi |
| **Pemulihan & Verifikasi** | One-time token SHA-256 hash            | Token acak 32-byte (TTL 15m/1h/24h) untuk reset password, verifikasi & ganti email |
| **Pluggable Mailer**       | Adapter Pattern (`internal/mailer`)    | `LogMailer` untuk dev lokal (terminal) & `ResendMailer` untuk production           |
| **Password Hashing**       | `github.com/alexedwards/argon2id`      | Standar hashing kredensial modern (RFC 9106)                                       |
| **Rate Limiter**           | `github.com/go-chi/httprate`           | Proteksi endpoint sensitif (login/register/forgot) berbasis IP (10 req/menit)      |
| **CORS**                   | `github.com/go-chi/cors`               | Konfigurasi origin via environment variable                                        |
| **Database Driver**        | `database/sql` + `go-sql-driver/mysql` | Kontrol penuh atas connection pool & transaksi                                     |
| **SQL Codegen**            | `sqlc` (`sqlc.yaml`)                   | Type-safe Go code dari query SQL murni tanpa overhead ORM                          |
| **Migrasi DB**             | `github.com/golang-migrate/migrate/v4` | File SQL di-embed ke binary via `//go:embed` dengan dukungan flag `-fresh`         |
| **Live Reload**            | Air (`.air.toml`)                      | Auto-rebuild saat development                                                      |

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
│   ├── migrations/                     # File migrasi SQL berurutan (000001_init_schema.* & 000002_rbac_and_admin.*)
│   └── queries/                        # File .sql mentah sebagai sumber generator sqlc
│       ├── auth_tokens.sql             # Query untuk tabel auth_tokens (verifikasi & reset)
│       ├── roles.sql                   # Query master roles
│       ├── permissions.sql             # Query master permissions
│       ├── role_permissions.sql        # Query pivot relasi role-permission
│       ├── user_roles.sql              # Query pivot multi-role & permissions per user
│       └── <domain>.sql
├── docs/
│   ├── ref/
│   │   └── architecture_reference.md   # Blueprint arsitektur ini
│   ├── starter_kit_specification.md    # Spesifikasi teknis & arsitektur proyek
│   ├── technical_specification.md      # Spesifikasi teknis modul & database schema
│   ├── learning_guide.md               # Panduan belajar arsitektur proyek
│   └── postman_collection.json         # Koleksi endpoint siap uji untuk Postman/Insomnia
├── internal/
│   ├── config/
│   │   └── config.go                   # Struct Config & loader env (dengan nilai fallback aman)
│   ├── database/
│   │   ├── database.go                 # Init koneksi DB, pool settings, ping, panggil migrate & seeder
│   │   ├── migrate.go                  # Runner golang-migrate (+ dukungan flag -fresh untuk reset dev)
│   │   └── seeder.go                   # Idempotent initial data seeder dengan ULID, default roles & admin
│   ├── handlers/
│   │   ├── auth_handler.go             # Handler login, register, refresh (RTR), logout, forgot/reset password, email verification
│   │   ├── user_handler.go             # Handler profil, avatar, username, status akun, email change
│   │   ├── role_handler.go             # Handler CRUD master roles, permissions, & sync role-permissions
│   │   ├── admin_user_handler.go       # Handler admin user management (search, filter, status, multi-role)
│   │   ├── admin_*_handler.go          # Handler endpoint terproteksi khusus role admin
│   │   ├── public_*_handler.go         # Handler endpoint publik
│   │   └── handlers_test.go            # Integration-style HTTP tests via httptest + in-memory repo
│   ├── mailer/
│   │   ├── mailer.go                   # Interface Mailer abstraksi pengiriman email
│   │   ├── log_mailer.go               # Driver development lokal (cetak box ASCII ke console)
│   │   ├── resend_mailer.go            # Driver production plug-and-play via Resend REST API
│   │   └── factory.go                  # Factory konstruktor memilih driver berdasarkan config
│   ├── middleware/
│   │   ├── auth.go                     # JWTAuthMiddleware, RequireRole, RequireRoles, RequirePermission (Wildcard *)
│   │   └── cors.go                     # Konfigurasi middleware CORS
│   ├── models/
│   │   ├── auth.go                     # DTO autentikasi, TokenResponse, RegisterInput, RefreshToken
│   │   ├── auth_token.go               # Entitas AuthToken & DTOs Forgot/Reset/Verify/ChangeEmail
│   │   ├── user.go                     # Entitas User, UserProfile, UserWithProfile, DTOs
│   │   ├── role.go                     # Entitas Role, Permission, & DTOs RBAC / Admin User
│   │   └── <entity>.go                 # Domain struct, Input DTO, & metode Validate()
│   ├── repository/
│   │   ├── sqlc/                       # Hasil generate otomatis dari `sqlc generate` (jangan edit manual)
│   │   ├── auth_token_repository.go    # Interface & implementasi penyimpanan token sementara
│   │   ├── role_repository.go          # Interface & implementasi CRUD role & role-permissions
│   │   ├── permission_repository.go    # Interface & implementasi master permission
│   │   ├── user_repository.go          # Interface & implementasi user, multi-role, query admin dinamis
│   │   └── <entity>_repository.go      # Interface repository, sentinel error, wrapper sqlc & dynamic query
│   ├── routes/
│   │   └── routes.go                   # Registrasi global middleware, health check, static files, & sub-routes
│   ├── service/
│   │   ├── auth_service.go             # Logika autentikasi, RTR, Reuse Detection, Argon2id, Forgot/Reset/Verify
│   │   ├── user_service.go             # Logika pengelolaan profil, avatar, username, email change request/confirm
│   │   ├── rbac_service.go             # Logika CRUD role, permissions, proteksi system roles
│   │   ├── admin_user_service.go       # Logika user management & critical safeguard Super Admin terakhir
│   │   ├── <domain>_service.go         # Domain service lainnya (posts, categories, markdown)
│   │   └── *_test.go                   # Unit test business logic dengan mock repository
│   └── utils/
│       ├── ulid.go                     # Generator & validator ULID 26-karakter
│       ├── jwt.go                      # Generate/Validate JWT access token (roles claims) & SHA-256 token hashing
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
[ routes + middleware ]     ──► JWT Bearer Auth / Role Guard / Permission Guard (Wildcard Bypass) / Rate Limiter
     │
     ▼
[ internal/handlers ]       ──► Validasi DTO input, mapping HTTP status, hybrid delivery (Cookie + JSON)
     │ (via Service Interface)
     ▼
[ internal/service ]        ──► Business rules, password hash Argon2id, token family, RBAC & Super Admin Safeguard
     │ (via Repository Interface)               │ (via Mailer Interface)
     ▼                                          ▼
[ internal/repository ]                         [ internal/mailer ]
     │ (Penugasan ULID & Error Mapping)              ├── LogMailer (Terminal Dev Box)
     ▼                                               └── ResendMailer (Production API)
[ internal/repository/sqlc ]
     │ (Kode compile sqlc)
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
4. **Alur Pemulihan & Verifikasi Akun (One-Time Token)**:
   - Menggunakan token acak 32-byte (64 hex string) yang disimpan dalam hash **SHA-256** di tabel `auth_tokens`.
   - **Anti User-Enumeration**: Endpoint `forgot-password` dan `resend-verification` selalu mengembalikan status konstan `200 OK` tanpa membocorkan status keberadaan email di database.
   - **Pencabutan Sesi Paksa**: Saat reset password berhasil, seluruh sesi refresh token aktif pengguna dicabut seketika (_forced session revocation_).
5. **Pluggable Mailer Architecture**:
   - Menerapkan adapter pattern: pengiriman email diabstraksi oleh interface `Mailer`.
   - Secara default menggunakan `LogMailer` untuk lingkungan dev lokal (100% tanpa setup eksternal).
   - Siap plug-and-play dengan `ResendMailer` via konfigurasi `.env`.
6. **Delivery Hybrid**:
   - Response JSON body: `{ access_token, refresh_token, token_type, expires_in }`.
   - Response Cookie: `HttpOnly = true`, `SameSite = Lax`, `Path = /api/v1/auth`.
7. **Password Hashing Argon2id**:
   - Selalu menggunakan `argon2id.CreateHash` dengan default OWASP parameter.
8. **Upload File Aman**:
   - Validasi format dan **magic bytes** konten aktual via `http.DetectContentType`.
   - Menggunakan stream rewind (`io.SeekStart`) sebelum menyalin konten.
9. **Granular Role-Based Access Control (RBAC)**:
   - Pola izin terstandarisasi dengan format `resource.action` (contoh: `users.read`, `roles.create`, `posts.update`).
   - Tabel pivot `user_roles` dan `role_permissions` mendukung satu pengguna memiliki banyak peran (_multi-role_).
   - **Super Admin Wildcard Bypass (`*`)**: Pengguna dengan role `super_admin` otomatis diloloskan oleh `PermissionMiddleware` tanpa perlu pengecekan izin ke database per request.
   - Access Token (JWT) menyematkan klaim array `roles: []string` untuk verifikasi cepat di sisi client maupun middleware.
10. **Admin User Management & Critical Safeguard**:
    - Pengelolaan pengguna komprehensif bagi administrator: pencarian kata kunci, filter status (`active`, `inactive`, `pending_verification`, `suspended`), filter role, dan pagination.
    - **Super Admin Safeguard**: Perlindungan mutlak di level service yang mencegah penonaktifan (`inactive`/`suspended`), penghapusan, atau pencabutan role `super_admin` dari Super Admin aktif terakhir (selalu mengembalikan HTTP 403 Forbidden).

---

## 5. Strategi Verifikasi & Testing

Sebelum melakukan commit, pastikan seluruh pengujian lolos:

```powershell
go build ./...
go test -v ./...
go vet ./...
```
