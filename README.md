# Go REST API Starter Kit

A production-grade, modular, and security-hardened RESTful API starter kit built with **Go (Golang 1.25+)** and **MySQL 8.0+**. Designed with Clean Architecture, Domain-Driven Design (DDD) principles, decentralized ULID identifiers, modern JWT + Refresh Token Rotation (RTR), granular Role-Based Access Control (RBAC), and zero-Redis native background concurrency.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Router](https://img.shields.io/badge/Router-Chi_v5-black?style=flat-square)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Database-MySQL_8.0+-4479A1?style=flat-square&logo=mysql&logoColor=white)](https://www.mysql.com/)
[![Query Generator](https://img.shields.io/badge/Query_Builder-SQLC-007acc?style=flat-square)](https://sqlc.dev/)
[![Migrations](https://img.shields.io/badge/Migrations-golang--migrate-orange?style=flat-square)](https://github.com/golang-migrate/migrate)
[![Security](https://img.shields.io/badge/Password_Hash-Argon2id-critical?style=flat-square)](https://datatracker.ietf.org/doc/html/rfc9106)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](https://opensource.org/licenses/MIT)

---

## Documentation Hub

Documentation is modularized and targeted for three distinct audiences:

```text
                                [ Documentation Hub ]
                                         │
     ┌───────────────────┬───────────────┴───────────────┬───────────────────┐
     ▼                   ▼                               ▼                   ▼
[ Stakeholder ]   [ Backend Dev ]                 [ Frontend Dev ]       [ DevOps ]
Overview & Arch   Onboarding, SQLC & Workers      Client RTR & Contracts Deploy & Cloud
```

| Audience / Concern | Key Guides & Specifications |
| :--- | :--- |
| **Stakeholder & System Overview** | - [System Architecture & Principles](docs/overview/system_architecture.md)<br>- [Security Architecture & Granular RBAC](docs/overview/security_and_rbac.md)<br>- [Data Dictionary & Entity Relationships](docs/overview/data_dictionary.md) |
| **Backend Developer & Contributor** | - [Developer Onboarding & Local Workflow](docs/backend/onboarding.md)<br>- [Database Engineering, SQLC, & Migrations](docs/backend/database_and_migrations.md)<br>- [Step-by-Step: Adding a New Module](docs/backend/how_to_add_new_module.md)<br>- [Zero-Redis Background Workers](docs/backend/background_workers.md)<br>- [Automated Testing Standards](docs/backend/testing_guide.md) |
| **Frontend Developer & API Consumer** | - [Client Integration & RTR Interceptor Guide](docs/frontend/client_integration_guide.md)<br>- [Response Envelope & Standard Error Codes](docs/frontend/response_and_error_codes.md)<br>- [Complete REST API Contract Directory](docs/frontend/api_contracts.md) |
| **DevOps & Production Operations** | - [Production Deployment & Infrastructure Guide](docs/devops/production_deployment.md) |
| **Testing Suites & Tools** | - [Postman Test Collection (71 Requests)](docs/tools/postman_collection.json)<br>- [Postman Testing Workflow Guide](docs/tools/postman_guide.md)<br>- [VS Code REST Client Suite](api.http) |

---

## Architectural Blueprint

The application enforces strict **Clean Architecture** and **Dependency Inversion**, isolating business rules from HTTP frameworks, persistence engines, and external vendors:

```text
[ Client HTTP Request ]
         │
         ▼
[ Chi Router & Global Middlewares ]
  ├── Request ID & RealIP Injection
  ├── Structured Logger (slog) & Panic Recovery
  ├── OWASP Security Headers & CORS Preflight
  └── In-Memory Rate Limiting (httprate)
         │
         ▼
[ Presentation Layer: HTTP Handlers (internal/handlers) ]
  ├── Payload Decoding (MaxBytesReader: 1MB JSON, 5MB Avatar, 10MB Media)
  └── Multi-Field Validation & Aggregated Errors (internal/models)
         │
         ▼
[ Domain & Business Layer (internal/service) ]
  ├── Domain Invariants & Argon2id Password Encryption
  ├── HTML UGC Sanitization (bluemonday) & Media Relationship Parsing
  ├── Refresh Token Rotation (RTR) & Reuse Detection Guard
  ├── Super Admin Protection Safeguard
  ├── Pluggable Storage (Local Disk vs Cloudflare R2 / AWS S3)
  └── Pluggable Mailer (Console Log vs Resend API)
         │
         ▼
[ Data Access Layer (internal/repository) ]
  ├── SQLC Type-Safe Parameterized Query Execution
  └── ULID Generator & Normalization (CHAR(26) Crockford Base32)
         │
         ▼
[ MySQL 8.0+ Database Engine (InnoDB) ]
```

---

## Key Features & Capabilities

- **Decentralized Entity Identity (ULID)**: All entities standardize on 26-character Crockford Base32 strings (`CHAR(26)`). Prevents ID enumeration attacks, guarantees chronological sorting (`ORDER BY id DESC`), and prevents JavaScript 53-bit integer truncation issues.
- **Enterprise Authentication (JWT + RTR)**:
  - Short-lived stateless Access Tokens (15 min).
  - Cryptographic stateful Refresh Tokens (7 days, SHA-256 hashed).
  - Token Family Lineage tracking with automatic full-family revocation upon reuse detection.
  - Hybrid client support: `HttpOnly`, `SameSite=Lax` cookies for Web SPAs alongside `X-Refresh-Token` headers for mobile/desktop.
  - Automated multi-device session severance (`RevokeAllForUser`) upon password change, email update, role alteration, or account deactivation.
- **Granular Role-Based Access Control (RBAC)**:
  - Multi-role assignments per user (`resource.action` dot syntax, e.g. `articles.create`, `media.upload`).
  - Wildcard bypass (`*`) for accounts holding the `super_admin` role.
  - High-performance thread-safe in-memory permission caching (`sync.Map`) with immediate cache eviction upon role mutations.
- **Administrative Invariant Safeguard**:
  - Hardware-level protection preventing the deletion, deactivation, suspension, or demotion of the last remaining active `super_admin`.
- **Zero-Redis Native Concurrency**:
  - Three periodic background goroutines managed via `time.Ticker` and `sync.WaitGroup` with zero external broker requirements:
    1. **Token Pruning Worker**: Cleans expired/revoked sessions every 24 hours.
    2. **Article Publisher Worker**: Transitions scheduled articles every 1 minute.
    3. **Analytics Aggregator & Retention Worker**: Aggregates daily engagement metrics and prunes raw events older than 30 days in bounded batches of 5,000 rows every 1 hour.
- **Multi-Input Field Validation**:
  - Comprehensive constraint evaluation collecting all payload errors simultaneously without early return.
- **Rich Media & Content Engine**:
  - `bluemonday` UGC HTML purification against XSS.
  - Native DOM tokenizer synchronizing embedded `<img data-media-id="...">` relations.
  - Historical snapshots created automatically in `article_revisions` on every update.
  - Privacy-preserving visitor analytics tracking via salted pseudonymized hashes (`SHA-256(IP + UserAgent + MonthlySalt)`).

---

## Technology Stack

| Layer / Concern | Technology | Purpose |
| :--- | :--- | :--- |
| **Language Runtime** | Go 1.25+ | High-performance compiled backend runtime |
| **HTTP Router** | Chi v5.2 | Lightweight, composable `net/http` router |
| **Database** | MySQL 8.0+ | Relational engine, InnoDB, `utf8mb4_0900_ai_ci` |
| **Schema Migrations** | golang-migrate v4.19 | Embedded migration runner (`db/embed.go`) |
| **Query Generator** | SQLC v1.27 | Compile-time type-safe Go SQL compiler |
| **Entity ID** | ULID v2.1 | Lexicographically sortable 128-bit unique identifiers |
| **Password Hashing** | Argon2id (RFC 9106) | Memory-hard key derivation function |
| **Rate Limiter** | httprate v0.15 | In-memory IP token-bucket rate limiting |
| **Object Storage** | AWS SDK for Go v2 | S3 & Cloudflare R2 zero-egress cloud storage |
| **HTML Sanitizer** | Bluemonday v1.0 | UGC-compliant HTML sanitization |
| **Dev Live Reload** | Air v1.62 | File watcher and binary rebuilder |

---

## Quick Start (3 Steps)

### 1. Configure Environment
```bash
cp .env.example .env
```
*(Adjust database credentials and secrets as needed)*

### 2. Initialize Database & Seed Administrator
Run the binary once with the `-fresh` flag to execute all migrations sequentially and seed initial taxonomies, roles, permissions, sample articles, and the default super administrator (`admin@mmdbconnect.com` / `adminpassword123`):
```bash
go run cmd/server/main.go -fresh
```

### 3. Launch Development Server
```bash
# With live-reload via Air:
air

# Or directly via Go:
go run cmd/server/main.go
```
The server binds to `http://localhost:8080`.

---

## API Testing & Verification

- **VS Code REST Client**: Open [`api.http`](api.http) to execute requests directly from your editor.
- **Postman**: Import [`docs/tools/postman_collection.json`](docs/tools/postman_collection.json) (71 preconfigured requests) and follow the [Postman Testing Guide](docs/tools/postman_guide.md).
- **Run Automated Test Suite**:
  ```bash
  go test -v -count=1 ./...
  go test -race ./...
  ```

---

## License

This project is licensed under the MIT License. This repository is released under the [MIT Open Source License](https://opensource.org/licenses/MIT).
