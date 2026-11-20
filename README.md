# Go REST API Starter Kit

A production-grade, modular, and security-hardened RESTful API starter kit built with **Go (Golang)** and **MySQL 8.0+**. Designed with clean architecture, enterprise authentication standards, granular role-based access control (RBAC), and pluggable infrastructure drivers.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Router](https://img.shields.io/badge/Router-Chi_v5-black?style=flat-square)](https://github.com/go-chi/chi)
[![Database](https://img.shields.io/badge/Database-MySQL_8.0+-4479A1?style=flat-square&logo=mysql&logoColor=white)](https://www.mysql.com/)
[![ORM / Query](https://img.shields.io/badge/Query_Builder-SQLC-007acc?style=flat-square)](https://sqlc.dev/)
[![Security](https://img.shields.io/badge/Password_Hash-Argon2id-critical?style=flat-square)](https://datatracker.ietf.org/doc/html/rfc9106)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

---

## Table of Contents

- [Go REST API Starter Kit](#go-rest-api-starter-kit)
  - [Table of Contents](#table-of-contents)
  - [Architectural Overview](#architectural-overview)
  - [Key Features \& Capabilities](#key-features--capabilities)
    - [1. Identity \& Data Consistency](#1-identity--data-consistency)
    - [2. Enterprise Authentication \& Session Management](#2-enterprise-authentication--session-management)
    - [3. Granular Role-Based Access Control (RBAC)](#3-granular-role-based-access-control-rbac)
    - [4. Administrative Safeguards \& Hardening](#4-administrative-safeguards--hardening)
    - [5. Multi-Input Field-Level Validation \& Unified Error Format](#5-multi-input-field-level-validation--unified-error-format)
    - [6. Pluggable Infrastructure Adapters](#6-pluggable-infrastructure-adapters)
    - [7. Zero-Redis Background Workers \& Hardening](#7-zero-redis-background-workers--hardening)
    - [8. Enterprise Article Management \& Analytics Engine](#8-enterprise-article-management--analytics-engine)
  - [Technology Stack](#technology-stack)
  - [Directory Layout](#directory-layout)
  - [Database Schema \& Data Models](#database-schema--data-models)
  - [API Reference](#api-reference)
    - [0. System Health \& Probes (`/health`)](#0-system-health--probes-health)
    - [1. Authentication (`/api/v1/auth`)](#1-authentication-apiv1auth)
    - [2. User Profile Management (`/api/v1/users`)](#2-user-profile-management-apiv1users)
    - [3. Public Articles \& Taxonomy (`/api/v1/articles`, `/categories`, `/tags`)](#3-public-articles--taxonomy-apiv1articles-categories-tags)
    - [4. Article \& Content Administration (`/api/v1/admin`)](#4-article--content-administration-apiv1admin)
    - [5. Role-Based Access Control (`/api/v1/admin/roles` \& `/permissions`)](#5-role-based-access-control-apiv1adminroles--permissions)
    - [6. User Administration (`/api/v1/admin/users`)](#6-user-administration-apiv1adminusers)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [1. Configuration Setup](#1-configuration-setup)
    - [2. Database Initialization](#2-database-initialization)
    - [3. Development Server](#3-development-server)
    - [4. Testing](#4-testing)
    - [5. Docker Environment](#5-docker-environment)
  - [Production \& Cloud Drivers](#production--cloud-drivers)
  - [Documentation \& Extended Guides](#documentation--extended-guides)
  - [API Testing \& Tooling](#api-testing--tooling)
  - [License](#license)

---

## Architectural Overview

This project follows Clean Architecture and Domain-Driven Design (DDD) principles. It isolates business rules from delivery mechanisms and third-party libraries, ensuring maintainability, testability, and deterministic behavior.

```text
[ Client HTTP Request ]
         │
         ▼
[ Chi Router & Global Middleware ]
  ├── Structured Logger (slog)
  ├── Recovery & Panic Handler
  ├── CORS Preflight / Headers
  └── RealIP & Proxy Sanitization
         │
         ▼
[ Route Group & Authorization Pipeline ]
  ├── Rate Limiter (IP-based, httprate)
  ├── JWT Authentication Middleware
  ├── Role Guard (RequireRole / RequireRoles)
  └── Permission Guard (PermissionMiddleware)
         │
         ▼
[ HTTP Handlers (internal/handlers) ]
  ├── Payload Decoding & JSON Parsing
  └── Field-Level Multi-Error Validation (internal/models)
         │
         ▼
[ Service Layer (internal/service) ]
  ├── Core Business Logic & Invariants
  ├── Password Hashing (Argon2id)
  ├── Pluggable Mailer (internal/mailer)
  └── Pluggable Storage (internal/storage)
         │
         ▼
[ Repository Layer (internal/repository) ]
  ├── SQLC Type-Safe Query Execution
  └── ULID Generator & Normalization (internal/utils)
         │
         ▼
[ MySQL 8.0+ Database Engine ]
```

---

## Key Features & Capabilities

### 1. Identity & Data Consistency

- **ULID Primary & Foreign Keys**: All entities use 26-character Crockford Base32 Universally Unique Lexicographically Sortable Identifiers (`CHAR(26)`). Prevents ID enumeration attacks, guarantees chronological sorting (`ORDER BY id DESC`), and eliminates 53-bit JavaScript integer truncation issues.

### 2. Enterprise Authentication & Session Management

- **JWT + Refresh Token Rotation (RTR)**:
  - **Access Token**: Stateless, short-lived (15 minutes default) HS256 JWT containing user identity and role claims.
  - **Refresh Token**: Cryptographically secure 64-character random string, stored as a SHA-256 hash in the database.
  - **Token Family & Reuse Detection**: Automatically tracks lineage. If an expired or revoked refresh token is reused (indicating session interception), the system revokes the entire token family immediately.
- **Automated Full-Session Revocation (`RevokeAllForUser`)**:
  - Automatically terminates all active client sessions across devices whenever security-sensitive actions occur: password change/reset, email change confirmation, user deactivation/suspension, role modifications/demotion, or account deletion.
- **Hybrid Delivery Strategy**: Supports browser-based Single Page Applications via `HttpOnly`, `SameSite=Lax` cookies, alongside mobile/desktop clients via standard `Authorization: Bearer <token>` headers, `X-Refresh-Token` header, and JSON payload bodies.

### 3. Granular Role-Based Access Control (RBAC)

- Multi-role association per user (`users` -> `user_roles` -> `roles`).
- **Dual-Layer Role Synchronization**: Modifying user roles updates relational `user_roles` assignments while simultaneously synchronizing `users.role` to prevent JWT identity claim divergence.
- Fine-grained permission model structured as `resource.action` (e.g., `users.read`, `roles.create`, `articles.create`, `articles.delete`, `tags.manage`, `media.upload`, `analytics.read`).
- Automatic wildcard bypass (`*`) for accounts holding the `super_admin` role.
- Modular middleware decorators: `RequireRole`, `RequireRoles`, and `PermissionMiddleware` with in-memory TTL caching and immediate cache eviction upon role/permission updates.

### 4. Administrative Safeguards & Hardening

- **Critical Super Admin Protection**: Built-in invariant logic prevents the last remaining active `super_admin` from being deleted, deactivated, suspended, or demoted, preventing irreversible system lockouts.
- **Route Parameter ULID Validation**: Standardized strict 26-character Crockford Base32 validation across all administrative and public route parameters, returning `INVALID_ID` on malformed inputs before database query execution.
- **OWASP Security Headers & HSTS**: Strict security headers injected on all HTTP responses (`nosniff`, `DENY` clickjacking, strict referrer, CSP) with automatic HTTP Strict Transport Security (`Strict-Transport-Security: max-age=31536000; includeSubDomains`) on TLS/HTTPS connections.

### 5. Multi-Input Field-Level Validation & Unified Error Format

- **Multi-Input Field-Level Validation**: Comprehensive payload evaluation collecting all field constraints into a unified `errors` map without early exit, enabling client-side forms to display inline feedback across all fields simultaneously:
  ```json
  {
    "code": "VALIDATION_FAILED",
    "error": "validation failed",
    "message": "validation failed",
    "errors": {
      "username": {
        "code": "TOO_SHORT",
        "message": "username must be 3-30 characters long"
      },
      "email": {
        "code": "INVALID_EMAIL",
        "message": "invalid email format"
      },
      "password": {
        "code": "TOO_SHORT",
        "message": "password must be at least 8 characters long"
      }
    }
  }
  ```
- **Unified Production-Grade Error Envelope**: All application errors across domain logic, authentication, permissions, rate limits, and 404/405 routing return a standardized JSON structure with machine-readable `code` alongside `error` and `message` properties:
  ```json
  {
    "code": "EMAIL_ALREADY_EXISTS",
    "error": "email is already registered",
    "message": "email is already registered"
  }
  ```

### 6. Pluggable Infrastructure Adapters

- **Storage Driver (`internal/storage`)**:
  - `local`: Writes to local filesystem (`./uploads`) with path traversal sanitization and Chi static file serving.
  - `s3` / `r2`: Cloudflare R2 (zero egress fees) and AWS S3 integration via AWS SDK for Go v2.
  - **Zero Leakage Renaming**: Uploaded files are renamed using pure ULIDs (`avatar_<ULID>.<ext>`), erasing original client filenames to eliminate path traversal vulnerabilities.
  - **Automated Orphan Cleanup**: Automatically purges replaced or deleted files from storage whenever users update avatars, remove covers, or delete accounts.
  - **Graceful Fallback**: Automatically reverts to local storage with warning logs if cloud credentials are incomplete.
- **Mailer Driver (`internal/mailer`)**:
  - `log`: Console dev mailer outputting formatted terminal inspection boxes without requiring network access or third-party accounts.
  - `resend`: Production-ready transactional email engine leveraging Resend's REST API for verification and password reset workflows.

### 7. Zero-Redis Background Workers & Hardening

- **Scheduled Article Publishing Worker**: Native Go goroutine running every 1 minute to automatically publish scheduled articles (`scheduled_at <= NOW()`).
- **Daily Analytics Aggregation & Pruning Worker**: Native Go goroutine running every 1 hour to aggregate daily engagement metrics using SARGable timestamp range scans and prune raw analytics events older than 30 days in bounded batches of 5,000 rows to prevent table lock contention.
- **Automated Token Pruning**: Background worker running every 24 hours to delete expired refresh tokens and consumed auth tokens.
- **Graceful Shutdown**: All background workers register with cancellation contexts and `sync.WaitGroup` to ensure clean exit upon receiving `SIGINT`/`SIGTERM`.
- **Rate Limiting**: Configured IP rate limiting (10 requests/minute for auth, 60 requests/minute for analytics) protecting sensitive endpoints against brute-force attacks.
- **Argon2id Hashing**: Password hashing compliant with OWASP guidelines and RFC 9106.

### 8. Enterprise Article Management & Analytics Engine

- **Strict HTML XSS Sanitization**: Content stored as clean HTML `LONGTEXT` purified using `bluemonday` UGC Policy before persistence.
- **Embedded Media Relationship Synchronization**: Native HTML tokenizer (`golang.org/x/net/html`) automatically extracts embedded `<img data-media-id="...">` tags and maintains transactional referential integrity in `article_media`.
- **Historical Revision Snapshots**: Every article update automatically creates a historical version snapshot in `article_revisions`.
- **High-Performance Composite Indexes**: Migration `000002_add_performance_indexes` provides specialized composite B-tree indexes for article filtering (`status, published_at, deleted_at`), daily analytics range queries, and retention pruning.
- **Media Asset Management**: Multipart image uploads with automatic dimension sniffing (`image.Config`), magic byte inspection, and referential protection preventing deletion of in-use media.
- **Privacy-Preserving Visitor Analytics**: GDPR-compliant visitor tracking using pseudonymized salted hashes (`SHA-256(IP + UserAgent + MonthlySalt)`) with zero cookie requirements.

---

## Technology Stack

| Layer / Concern       | Technology        | Version | Purpose                                               |
| :-------------------- | :---------------- | :------ | :---------------------------------------------------- |
| Runtime Language      | Go (Golang)       | 1.25+   | High-performance compiled backend runtime             |
| HTTP Router           | Chi Router        | v5.2.5  | Lightweight, idiomatic `net/http` subrouting          |
| Primary Storage       | MySQL             | 8.0+    | Relational engine, InnoDB, `utf8mb4_0900_ai_ci`       |
| Database Migration    | golang-migrate    | v4.19.1 | Embedded DDL migration runner (`iofs` / `embed`)      |
| Query Generator       | SQLC              | v1.27.0 | Compile-time type-safe Go repository generation       |
| Entity Identification | ULID              | v2.1.0  | Lexicographically sortable 128-bit unique identifiers |
| Authentication        | Golang JWT        | v5.3.1  | Stateless JSON Web Tokens (HS256)                     |
| Password Encryption   | Argon2id          | -       | Modern memory-hard key derivation                     |
| Rate Limiter          | httprate          | v0.15.1 | In-memory token bucket rate limiting                  |
| Cloud Object Storage  | AWS SDK for Go v2 | v2.x    | S3 / Cloudflare R2 API compatibility                  |
| HTML Sanitizer        | Bluemonday        | v1.0.27 | UGC-compliant HTML sanitization for article content   |
| HTML Parser           | x/net/html        | -       | DOM tokenizer for embedded media synchronization      |
| Hot Reloading         | Air               | v1.62.0 | Development file watcher and binary rebuilder         |

---

## Directory Layout

```text
.
├── cmd/
│   └── server/
│       └── main.go                  # Application entrypoint & composition root
├── db/
│   ├── embed.go                     # Migration filesystem embed directive
│   ├── migrations/                  # Schema DDL versions (up/down SQL)
│   └── queries/                     # Raw SQL statements consumed by SQLC
├── docs/
│   ├── architecture.md          # Architectural blueprint, layering, and design principles
│   ├── how_to_add_new_module.md # Step-by-step practical guide: Adding new CRUD features
│   ├── security_and_rbac.md     # Authentication (JWT RTR), RBAC, and security safeguards
│   ├── deployment.md            # Production deployment, Docker, Resend, and Cloudflare R2
│   ├── learning_guide.md        # Educational walkthrough of Go concepts and idioms
│   ├── postman_collection.json  # Postman v2.1 test collection
│   └── specs/                   # Historical build-phase technical specifications
├── internal/
│   ├── config/                      # Environment variables loading and validation
│   ├── database/                    # MySQL connection pooling, migrations, & seeder
│   ├── handlers/                    # HTTP presentation layer & request controllers
│   ├── mailer/                      # Pluggable transactional email adapters
│   ├── middleware/                  # Auth, RBAC, Rate Limit, and CORS middleware
│   ├── models/                      # Domain entities, request DTOs, & validators
│   ├── repository/                  # Data access layer & SQLC interfaces
│   ├── routes/                      # Route registry and middleware tree assembly
│   ├── service/                     # Core business logic layer
│   ├── storage/                     # Pluggable local and cloud storage adapters
│   └── utils/                       # Cryptographic tokens, ULID, JWT, & JSON helpers
├── .air.toml                        # Live reload configuration
├── .env.example                     # Environment variables template
├── api.http                         # VS Code REST Client executable requests
├── compose.yaml                     # Production multi-container Docker configuration
├── Dockerfile                       # Multi-stage lightweight scratch/alpine build
└── sqlc.yaml                        # SQLC generator rules and mapping configuration
```

---

## Database Schema & Data Models

All primary and relational foreign keys utilize `CHAR(26)` UTF-8 strings representing ULID values (with `BIGINT AUTO_INCREMENT` used strictly for high-frequency internal analytics events).

| Table                      | Purpose                                        | Key Attributes                                                                                                                 |
| :------------------------- | :--------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------- |
| `users`                    | Core user identity & credentials               | `id`, `username`, `email`, `password_hash`, `role`, `status`, `email_verified_at`                                              |
| `user_profiles`            | Extended personal profile data                 | `user_id` (PK/FK), `full_name`, `bio`, `phone_number`, `avatar_url`                                                            |
| `refresh_tokens`           | Stateful session lifecycle tracking            | `id`, `user_id`, `token_hash`, `token_family`, `is_revoked`, `expires_at`                                                      |
| `auth_tokens`              | Ephemeral tokens (Verify, Reset, Email Change) | `id`, `user_id`, `token_hash`, `token_type`, `payload`, `is_used`, `expires_at`                                                |
| `roles`                    | RBAC role definitions                          | `id`, `name`, `display_name`, `description`                                                                                    |
| `permissions`              | RBAC granular permissions                      | `id`, `name`, `description` (format: `resource.action`)                                                                        |
| `role_permissions`         | Pivot table for role-to-permission mapping     | `role_id`, `permission_id`                                                                                                     |
| `user_roles`               | Pivot table for user multi-role assignment     | `user_id`, `role_id`                                                                                                           |
| `categories`               | Article classification blueprint               | `id`, `name`, `slug`, `description`                                                                                            |
| `tags`                     | Tag taxonomy blueprint                         | `id`, `name`, `slug`                                                                                                           |
| `media`                    | Centralized media asset storage catalog        | `id`, `uploaded_by`, `filename`, `storage_disk`, `storage_key`, `url`, `mime_type`, `size`, `width`, `height`                  |
| `articles`                 | Article content management core                | `id`, `author_id`, `category_id`, `featured_image_id`, `og_image_id`, `title`, `slug`, `excerpt`, `content` (HTML), `status`   |
| `article_tags`             | Pivot table for article many-to-many tags      | `article_id`, `tag_id`                                                                                                         |
| `article_media`            | Pivot table for embedded media tracking        | `article_id`, `media_id`                                                                                                       |
| `article_revisions`        | Full version snapshot history per update       | `id`, `article_id`, `author_id`, `title`, `slug`, `excerpt`, `content`, `revision_number`                                      |
| `article_related`          | Manual curated related articles                | `article_id`, `related_article_id`                                                                                             |
| `article_analytics_events` | Raw GDPR-compliant visitor engagement events   | `id`, `article_id`, `event_id`, `visitor_id` (salted hash), `session_id`, `event_type`, `reading_time_seconds`, `scroll_depth` |
| `article_daily_analytics`  | Aggregated daily engagement metrics            | `id`, `article_id`, `analytics_date`, `total_views`, `unique_visitors`, `total_shares`, `total_reading_time`                   |
| `settings`                 | Global application configuration key-value     | `key_name`, `value_text`                                                                                                       |

---

## API Reference

### 0. System Health & Probes (`/health`)

| Method | Endpoint        | Authorization | Description                                                                        |
| :----- | :-------------- | :------------ | :--------------------------------------------------------------------------------- |
| `GET`  | `/health`       | Public        | Deep health probe with MySQL connection verification (returns 200 OK / 503 error). |
| `GET`  | `/health/live`  | Public        | Instant liveness probe for container lifecycle managers (Kubernetes / Docker).     |
| `GET`  | `/health/ready` | Public        | Readiness probe for ingress traffic routing.                                       |

### 1. Authentication (`/api/v1/auth`)

| Method | Endpoint                           | Authorization         | Description                                                                      |
| :----- | :--------------------------------- | :-------------------- | :------------------------------------------------------------------------------- |
| `POST` | `/api/v1/auth/register`            | Public (Rate Limited) | Registers a new account and dispatches an email verification token.              |
| `POST` | `/api/v1/auth/login`               | Public (Rate Limited) | Authenticates user; returns access token (JSON) and refresh token (Cookie/JSON). |
| `POST` | `/api/v1/auth/refresh`             | Public / Cookie       | Rotates refresh token and issues a new access token via RTR.                     |
| `POST` | `/api/v1/auth/logout`              | Bearer Token          | Revokes refresh token session and invalidates HTTP-only cookie.                  |
| `POST` | `/api/v1/auth/change-password`     | Bearer Token          | Updates password and terminates all other active sessions.                       |
| `POST` | `/api/v1/auth/forgot-password`     | Public (Rate Limited) | Dispatches reset token. Anti-enumeration returns 200 OK unconditionally.         |
| `POST` | `/api/v1/auth/reset-password`      | Public (Rate Limited) | Validates reset token and sets new password; revokes all sessions.               |
| `POST` | `/api/v1/auth/verify-email`        | Public (Rate Limited) | Confirms registration via query parameter (`?token=...`) or JSON body.           |
| `POST` | `/api/v1/auth/resend-verification` | Public (Rate Limited) | Resends verification email. Anti-enumeration returns 200 OK unconditionally.     |

### 2. User Profile Management (`/api/v1/users`)

| Method   | Endpoint                             | Authorization         | Description                                                                   |
| :------- | :----------------------------------- | :-------------------- | :---------------------------------------------------------------------------- |
| `GET`    | `/api/v1/users/me`                   | Bearer Token          | Retrieves authenticated user account and profile data.                        |
| `PUT`    | `/api/v1/users/profile`              | Bearer Token          | Updates profile details (`full_name`, `bio`, `phone_number`).                 |
| `PUT`    | `/api/v1/users/username`             | Bearer Token          | Updates username with uniqueness and character pattern validation.            |
| `GET`    | `/api/v1/users/status`               | Bearer Token          | Checks account verification and status.                                       |
| `POST`   | `/api/v1/users/avatar`               | Bearer Token          | Uploads profile picture (multipart form, magic bytes validation, max 5MB).    |
| `DELETE` | `/api/v1/users/avatar`               | Bearer Token          | Removes avatar file and resets database reference.                            |
| `POST`   | `/api/v1/users/email/request-change` | Bearer Token          | Initiates email change with current password check; sends confirmation token. |
| `POST`   | `/api/v1/users/email/confirm-change` | Public (Rate Limited) | Confirms email change via query parameter (`?token=...`) or JSON body.        |

### 3. Public Articles & Taxonomy (`/api/v1/articles`, `/categories`, `/tags`)

| Method | Endpoint                             | Authorization | Description                                                                                   |
| :----- | :----------------------------------- | :------------ | :-------------------------------------------------------------------------------------------- |
| `GET`  | `/api/v1/articles`                   | Public        | Lists published articles with filters (`category`, `tag`, `is_featured`, `search`, `page`).   |
| `GET`  | `/api/v1/articles/{slug}`            | Public        | Retrieves published article details by slug with category, tags, and media relations.         |
| `POST` | `/api/v1/articles/{id}/analytics`    | Public        | Ingests visitor engagement event (`view`, `read`, `scroll`, `share`) with idempotency header. |
| `GET`  | `/api/v1/categories`                 | Public        | Lists all public categories with article count.                                               |
| `GET`  | `/api/v1/categories/{slug}/articles` | Public        | Lists published articles under a specific category slug.                                      |
| `GET`  | `/api/v1/tags`                       | Public        | Lists all public tags with article count.                                                     |
| `GET`  | `/api/v1/tags/{slug}`                | Public        | Retrieves tag details by slug.                                                                |
| `GET`  | `/articles/{slug}`                   | Public        | SEO vanity route directly mapping to article details.                                         |

### 4. Article & Content Administration (`/api/v1/admin`)

| Method   | Endpoint                                | Permission Required | Description                                                                       |
| :------- | :-------------------------------------- | :------------------ | :-------------------------------------------------------------------------------- |
| `GET`    | `/api/v1/admin/articles`                | `articles.read`     | Lists all articles across all statuses (draft, scheduled, published, archived).   |
| `POST`   | `/api/v1/admin/articles`                | `articles.create`   | Creates a new article with XSS HTML sanitization and automated tag/media sync.    |
| `GET`    | `/api/v1/admin/articles/{id}`           | `articles.read`     | Retrieves article details by ULID.                                                |
| `PUT`    | `/api/v1/admin/articles/{id}`           | `articles.update`   | Updates article and automatically creates a snapshot in `article_revisions`.      |
| `DELETE` | `/api/v1/admin/articles/{id}`           | `articles.delete`   | Soft-deletes article (`deleted_at = NOW()`).                                      |
| `GET`    | `/api/v1/admin/articles/{id}/revisions` | `articles.read`     | Retrieves historical revision snapshots for an article.                           |
| `GET`    | `/api/v1/admin/categories`              | `categories.manage` | Lists all categories with metadata and article counts.                            |
| `POST`   | `/api/v1/admin/categories`              | `categories.manage` | Creates a new category.                                                           |
| `PUT`    | `/api/v1/admin/categories/{id}`         | `categories.manage` | Updates category name, slug, or description by ULID.                              |
| `DELETE` | `/api/v1/admin/categories/{id}`         | `categories.manage` | Deletes category (protected if category currently holds articles, HTTP 409).      |
| `GET`    | `/api/v1/admin/tags`                    | `tags.read`         | Lists all tags.                                                                   |
| `POST`   | `/api/v1/admin/tags`                    | `tags.manage`       | Creates a new tag.                                                                |
| `PUT`    | `/api/v1/admin/tags/{id}`               | `tags.manage`       | Updates tag name or slug by ULID.                                                 |
| `DELETE` | `/api/v1/admin/tags/{id}`               | `tags.manage`       | Deletes tag by ULID.                                                              |
| `POST`   | `/api/v1/admin/media`                   | `media.upload`      | Uploads image asset (JPG, PNG, WEBP, GIF, max 10MB) with auto dimension sniffing. |
| `GET`    | `/api/v1/admin/media`                   | `media.read`        | Lists media library assets with pagination.                                       |
| `GET`    | `/api/v1/admin/media/{id}`              | `media.read`        | Retrieves media metadata by ULID.                                                 |
| `DELETE` | `/api/v1/admin/media/{id}`              | `media.delete`      | Deletes media asset (protected if media is currently used in articles, HTTP 409). |
| `GET`    | `/api/v1/admin/articles/{id}/analytics` | `analytics.read`    | Retrieves daily engagement analytics report for an article by date range.         |
| `GET`    | `/api/v1/admin/analytics/overview`      | `analytics.read`    | Retrieves global total engagement analytics metrics across the platform.          |

### 5. Role-Based Access Control (`/api/v1/admin/roles` & `/permissions`)

| Method   | Endpoint                               | Permission Required  | Description                                      |
| :------- | :------------------------------------- | :------------------- | :----------------------------------------------- |
| `GET`    | `/api/v1/admin/roles`                  | `roles.read`         | Lists all configured roles.                      |
| `POST`   | `/api/v1/admin/roles`                  | `roles.create`       | Creates a new role.                              |
| `GET`    | `/api/v1/admin/roles/{id}`             | `roles.read`         | Retrieves role details and assigned permissions. |
| `PUT`    | `/api/v1/admin/roles/{id}`             | `roles.update`       | Updates role metadata.                           |
| `PUT`    | `/api/v1/admin/roles/{id}/permissions` | `roles.update`       | Synchronizes permission array for a role.        |
| `DELETE` | `/api/v1/admin/roles/{id}`             | `roles.delete`       | Deletes role by ULID.                            |
| `GET`    | `/api/v1/admin/permissions`            | `permissions.read`   | Lists all available system permissions.          |
| `POST`   | `/api/v1/admin/permissions`            | `permissions.create` | Registers a new permission descriptor.           |
| `GET`    | `/api/v1/admin/permissions/{id}`       | `permissions.read`   | Retrieves permission details.                    |
| `PUT`    | `/api/v1/admin/permissions/{id}`       | `permissions.update` | Updates permission description.                  |
| `DELETE` | `/api/v1/admin/permissions/{id}`       | `permissions.delete` | Deletes permission by ULID.                      |

### 6. User Administration (`/api/v1/admin/users`)

| Method   | Endpoint                          | Permission Required | Description                                                                 |
| :------- | :-------------------------------- | :------------------ | :-------------------------------------------------------------------------- |
| `GET`    | `/api/v1/admin/users`             | `users.read`        | Lists users (search, status filter, role filter, pagination).               |
| `GET`    | `/api/v1/admin/users/{id}`        | `users.read`        | Retrieves user profile and multi-role assignments.                          |
| `PUT`    | `/api/v1/admin/users/{id}`        | `users.update`      | Updates user credentials and profile information.                           |
| `PUT`    | `/api/v1/admin/users/{id}/status` | `users.update`      | Updates status (`active`, `inactive`, `pending_verification`, `suspended`). |
| `PUT`    | `/api/v1/admin/users/{id}/roles`  | `users.update`      | Assigns multi-roles to user.                                                |
| `DELETE` | `/api/v1/admin/users/{id}`        | `users.delete`      | Permanently deletes user and cleans up avatar from storage.                 |

_Note: Endpoints affecting user status, role assignments, and deletions enforce the Super Admin Safeguard, blocking modifications against the last active `super_admin` with HTTP 403 Forbidden._

---

## Getting Started

### Prerequisites

- **Go**: 1.25 or higher
- **MySQL Server**: 8.0 or higher
- **Make** or **Air** (optional, for hot-reloading)

### 1. Configuration Setup

Copy the environment template to create your local `.env`:

```bash
cp .env.example .env
```

Adjust the parameters to match your local database and secret keys:

```ini
PORT=8080
APP_ENV=development
FRONTEND_URL=http://localhost:3000

# Database Configuration & Pool Settings
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=go_crud_api
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME_MINUTES=60
DB_CONN_MAX_IDLE_TIME_MINUTES=10

# Initial Administrator Credentials (used during -fresh seed)
ADMIN_NAME=Administrator
ADMIN_EMAIL=admin@mmdbconnect.com
ADMIN_PASSWORD=adminpassword123

# JWT Token Lifecycles
JWT_SECRET=replace_with_a_long_random_entropy_secret_key
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Pluggable Mailer: 'log' (local terminal) or 'resend' (production)
MAILER_DRIVER=log
RESEND_API_KEY=
MAIL_FROM=Go Starter Kit <onboarding@resend.dev>

# Pluggable Storage: 'local', 'r2', or 's3'
STORAGE_DRIVER=local
UPLOAD_DIR=./uploads
```

### 2. Database Initialization

Execute the server binary with the `-fresh` flag once to drop any legacy tables, execute all migrations sequentially, and seed initial roles, permissions, categories, sample posts, and the default administrator account:

```bash
go run cmd/server/main.go -fresh
```

### 3. Development Server

Start the application with live reload using **Air**:

```bash
air
```

Alternatively, run directly via the Go toolchain:

```bash
go run cmd/server/main.go
```

The server binds to `http://localhost:8080`.

### 4. Testing

Run the full automated test suite (including validation, auth lifecycles, and storage tests):

```bash
go test -v -count=1 ./...
```

Run test suite with race condition detection:

```bash
go test -race ./...
```

### 5. Docker Environment

The repository includes a ready-to-run containerized setup with multi-stage build:

```bash
# Start MySQL and the Go REST API container
docker compose up -d --build

# Inspect container logs
docker compose logs -f api
```

---

## Production & Cloud Drivers

The starter kit features pluggable infrastructure adapters, allowing zero-code switching between local development and cloud production via `.env`:

| Capability               | Local Driver (`development`)         | Production Driver (`production`)                    | Documentation Guide                                                                                                                |
| :----------------------- | :----------------------------------- | :-------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------- |
| **Transactional Email**  | `MAILER_DRIVER=log` (Console Box)    | `MAILER_DRIVER=resend` (Resend API)                 | [docs/deployment.md#3-transactional-email-setup-resend](docs/deployment.md#3-transactional-email-setup-resend)                     |
| **Media & File Storage** | `STORAGE_DRIVER=local` (`./uploads`) | `STORAGE_DRIVER=r2` / `s3` (Cloudflare R2 / AWS S3) | [docs/deployment.md#4-object-storage-setup-cloudflare-r2--aws-s3](docs/deployment.md#4-object-storage-setup-cloudflare-r2--aws-s3) |

For complete production deployment instructions, Docker Compose setup, reverse proxy configurations (Nginx/Caddy), and security checklists, refer to [docs/deployment.md](docs/deployment.md).

---

## Documentation & Extended Guides

- **Adding New Modules**: Read the practical walkthrough on building new features in [docs/how_to_add_new_module.md](docs/how_to_add_new_module.md).
- **Postman Testing Guide**: Complete step-by-step testing walkthrough with sequential workflows and automated scripts in [docs/postman_testing_guide.md](docs/postman_testing_guide.md).
- **System Architecture**: Read the architectural design principles and layer breakdown in [docs/architecture.md](docs/architecture.md).
- **Security & Authorization**: In-depth details on JWT RTR, session reuse detection, and RBAC in [docs/security_and_rbac.md](docs/security_and_rbac.md).
- **Production Deployment**: Guide for Docker, Resend email, and Cloudflare R2 / AWS S3 in [docs/deployment.md](docs/deployment.md).
- **Go Learning Walkthrough**: Educational guide covering Go concepts and idioms used throughout the project in [docs/learning_guide.md](docs/learning_guide.md).
- **Historical Specifications**: Build-phase specifications are archived in [docs/specs/initial_system_specification.md](docs/specs/initial_system_specification.md).

---

## API Testing & Tooling

- **Postman Testing Guide & Collection**: Import [postman_collection.json](docs/postman_collection.json) and follow the step-by-step testing instructions in [docs/postman_testing_guide.md](docs/postman_testing_guide.md). Features automated post-response test scripts that automatically capture and inject Bearer tokens into your collection environment.
- **VS Code REST Client**: Open [api.http](api.http) to send requests directly from your code editor. Variables (`@accessToken`, `@refreshToken`, `@sampleToken`) allow complete step-by-step end-to-end testing without external applications.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for full details.
