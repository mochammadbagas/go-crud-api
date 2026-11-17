# Production-Grade Audit & Remediation Roadmap

This document outlines the findings and implementation roadmap resulting from the senior architecture and security audit of the `go-crud-api` backend starter kit.

---

## Audit Summary

The starter kit already implements a robust foundation:
- Secure identifier generation via ULID.
- High-security password hashing with Argon2id.
- Type-safe SQL query generation via SQLC with prepared statements.
- Structured REST routing with Chi v5.
- Unified response envelope with standardized `code` + `message` error formats.
- Clean shutdown handling with graceful context timeouts.

To achieve complete industry enterprise and production-readiness, seven enhancement items have been identified across three execution phases.

---

## Phase Breakdown

### Phase 1: HTTP Security & Denial of Service (DoS) Hardening

| Item | Problem Statement | Production-Grade Target | Target Components |
| :--- | :--- | :--- | :--- |
| **1.1 Request Body Size Limit** | Handlers parse unbounded JSON streams via `json.NewDecoder(r.Body)`. A malicious payload of hundreds of megabytes could cause memory exhaustion (OOM DoS). | Wrap JSON body decoding with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MB payload limit) and return a structured `PAYLOAD_TOO_LARGE` error if exceeded. | `internal/utils/request.go`, Handlers |
| **1.2 Static File Server Directory Browsing Prevention** | The `/uploads/*` route uses Go's default `http.FileServer`. Direct GET requests to directory paths can render directory listing HTML. | Wrap the filesystem with a restricted `http.FileSystem` implementation that returns `os.ErrNotExist` for directory requests lacking `index.html`. | `internal/routes/routes.go` |
| **1.3 OWASP Security Headers Middleware** | API responses currently only provide CORS headers. Standard browser and client protection headers are missing. | Inject standard OWASP security headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, and `X-XSS-Protection: 1; mode=block`. | `internal/middleware/security.go`, `internal/routes/routes.go` |

---

### Phase 2: Data Integrity & Database Resilience

| Item | Problem Statement | Production-Grade Target | Target Components |
| :--- | :--- | :--- | :--- |
| **2.1 Atomic Registration via Database Transaction** | In `AuthService.Register`, creating the user, creating their profile (`profileRepo.Create`), and assigning their default role (`userRepo.AssignRole`) are separate non-transactional database calls with discarded errors. A transient database failure during profile creation creates an orphaned user record. | Execute user creation, profile initialization, and role assignment within a database transaction (`sql.Tx`) or provide transaction orchestration at repository/service boundaries. | `internal/service/auth_service.go`, `internal/repository/` |
| **2.2 Connection Pool Idle Lifetime Tuning** | `internal/database` sets `MaxIdleConns`, `MaxOpenConns`, and `ConnMaxLifetime`, but omits `SetConnMaxIdleTime`. Cloud firewalls and MySQL `wait_timeout` can silently sever idle connections, causing intermittent "broken pipe" errors. | Set `db.SetConnMaxIdleTime(10 * time.Minute)` to gracefully reap idle connections before remote network or server dropouts. | `internal/database/database.go` |

---

### Phase 3: Operational Safety & Code Hygiene

| Item | Problem Statement | Production-Grade Target | Target Components |
| :--- | :--- | :--- | :--- |
| **3.1 Production JWT Secret Validation** | `config.go` provides a default fallback secret for local convenience. If deployed to production with `APP_ENV=production` without explicitly setting `JWT_SECRET`, the API runs insecurely without warning. | Validate `JWTSecret` during initialization: reject startup or log critical errors if default or weak secrets (< 32 characters) are detected in production environments. | `internal/config/config.go` |
| **3.2 Language Consistency in Core Errors and Logs** | A few residual Indonesian strings remain in core infrastructure files (`config.go` info log, `jwt.go` sentinel error, `ulid.go` sentinel error). | Standardize all core internal logs and sentinel errors to concise English technical descriptions. | `internal/config/config.go`, `internal/utils/jwt.go`, `internal/utils/ulid.go` |

---

## Status Tracker

- [x] **Phase 1: HTTP Security & Denial of Service (DoS) Hardening**
  - [x] 1.1 Request Body Size Limit (`http.MaxBytesReader` via `utils.DecodeJSON`)
  - [x] 1.2 Prevent Directory Listing on Static File Server (`noDirListingFS`)
  - [x] 1.3 OWASP Security Headers Middleware (`middleware.SecurityHeaders()`)
- [ ] **Phase 2: Data Integrity & Database Resilience**
  - [ ] 2.1 Atomic User Registration (`sql.Tx` / Transaction Boundary)
  - [ ] 2.2 Database Pool `SetConnMaxIdleTime`
- [ ] **Phase 3: Operational Safety & Code Hygiene**
  - [ ] 3.1 Strict JWT Secret Validation in Production Mode
  - [ ] 3.2 Standardize Residual Internal Strings to English
