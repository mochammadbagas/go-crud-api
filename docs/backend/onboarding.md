# Backend Developer Onboarding & Local Workflow

**Document Type:** Contributor Guide & Engineering Standards  
**Target Audience:** Backend Developers, Go Contributors, and DevOps Engineers  
**Prerequisites:** Go 1.25+, MySQL 8.0+, Git

---

## 1. Local Environment Prerequisites

Ensure you have the following toolchains installed on your host machine:

| Tool               | Minimum Version | Purpose                           | Installation / Verification                                                        |
| :----------------- | :-------------- | :-------------------------------- | :--------------------------------------------------------------------------------- |
| **Go**             | 1.25+           | Runtime & standard library        | `go version`                                                                       |
| **MySQL**          | 8.0+            | Relational persistence            | `mysql --version` (or run via Docker)                                              |
| **Air**            | v1.62+          | Live reloading during development | `go install github.com/air-verse/air@latest`                                       |
| **SQLC**           | v1.27+          | Compile SQL queries to Go structs | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`                              |
| **golang-migrate** | v4.19+          | Schema migrations CLI (optional)  | `go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

---

## 2. Configuration Setup

Copy the example configuration file to create your local `.env`:

```bash
cp .env.example .env
```

Review and customize the environment variables:

```ini
PORT=8080
APP_ENV=development
FRONTEND_URL=http://localhost:3000

# Database Credentials
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_local_password
DB_NAME=go_crud_api

# Connection Pool Defaults (Optimized for Dev)
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME_MINUTES=60
DB_CONN_MAX_IDLE_TIME_MINUTES=10

# Default Administrator (Seeded on -fresh)
ADMIN_NAME=Administrator
ADMIN_EMAIL=admin@mmdbconnect.com
ADMIN_PASSWORD=adminpassword123

# Security & Tokens
JWT_SECRET=development_super_secret_key_at_least_32_chars_long
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Pluggable Adapters (Local Defaults)
MAILER_DRIVER=log          # 'log' outputs emails to terminal
STORAGE_DRIVER=local       # 'local' writes to ./uploads
UPLOAD_DIR=./uploads
```

---

## 3. Database Initialization & Seeding

The server binary includes an embedded migration runner (`db/embed.go`) using `golang-migrate`.

To wipe any existing local tables, execute all DDL migrations sequentially, and seed initial roles, permissions, categories, sample articles, and the default super administrator account:

```bash
go run cmd/server/main.go -fresh
```

> [!WARNING]
> Never run with `-fresh` in a production environment as it drops all existing tables before migrating.

To run the application normally without reseeding:

```bash
go run cmd/server/main.go
```

---

## 4. Hot Reloading with Air

For an optimized developer experience with automatic compilation on file changes:

```bash
air
```

Air monitors `.go`, `.sql`, and `.html` files based on `.air.toml` configuration and rebuilds the binary in `tmp/main.exe`.

---

## 5. Running Tests

Run the complete automated test suite (including validation, auth, and storage tests):

```bash
go test -v -count=1 ./...
```

Run tests with Go's race condition detector:

```bash
go test -race ./...
```

Generate a test coverage profile:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## 6. Code Style & Architecture Conventions

1. **Clean Architecture Isolation**:
   - `internal/models`: Domain models and multi-field validation logic (`input.Validate()`). No HTTP or database logic.
   - `internal/repository`: Database query definitions and interfaces.
   - `internal/service`: Core business rules, invariant validations, and domain orchestration.
   - `internal/handlers`: HTTP decoding, routing logic, and JSON response formatting.
2. **Sentinel Errors**: Always define exported sentinel errors in the service or repository layer (e.g. `var ErrNotFound = errors.New("...")`) and compare them in handlers using `errors.Is(err, target)`.
3. **Pointers for Nullable Fields**: Use standard Go pointers (`*string`, `*time.Time`) for nullable database columns to map cleanly to SQL `NULL` and JSON `null` / `omitempty`.
4. **ULID Primary Keys**: Every new entity must use a 26-character Crockford Base32 ULID generated via `utils.NewULID()`.
