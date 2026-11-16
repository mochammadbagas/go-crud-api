# System Architecture & Design Principles

**Document Type:** Core Architectural Reference & Engineering Standards  
**Target Audience:** Software Engineers, Backend Developers, and System Architects  
**Standard:** Enterprise Clean Architecture, Dependency Inversion, Domain-Driven Design (DDD)

---

## 1. Architectural Philosophy

The **Go REST API Starter Kit** is structured around the principles of **Clean Architecture**. The primary objective is separation of concerns: isolating core business rules from delivery mechanisms (HTTP/REST), external infrastructure (MySQL, S3, Resend), and third-party frameworks.

```text
               +---------------------------------------+
               |             HTTP Delivery             |
               |       (Chi Router, Middleware)        |
               +-------------------+-------------------+
                                   |
                                   v
               +-------------------+-------------------+
               |            Controllers                |
               |        (internal/handlers)            |
               +-------------------+-------------------+
                                   | (Service Interface)
                                   v
               +-------------------+-------------------+
               |           Domain Logic                |
               |         (internal/service)            |
               +---------+-------------------+---------+
                         |                   |
    (Storage Interface)  |                   | (Repository Interface)
                         v                   v
               +---------+---------+   +-----+---------+
               | Pluggable Storage |   |  Repository   |
               |   & Mailer SDK    |   |  (SQLC/MySQL) |
               +-------------------+   +---------------+
```

### Key Architectural Tenets:

1. **Independent of Frameworks**: Business logic does not import Chi, HTTP headers, or request contexts. It can be consumed by CLI tools, gRPC handlers, or background jobs without modification.
2. **Testable in Isolation**: Business rules can be validated with unit tests using in-memory mock repositories without spinning up a live MySQL instance or external network connections.
3. **Independent of Database**: Repositories expose Go interfaces. The underlying persistence mechanism (MySQL via SQLC) can be refactored without altering services.
4. **Compile-Time Type Safety**: ORMs using runtime reflection are avoided in favor of **SQLC**, which compiles raw SQL into strongly-typed Go methods.

---

## 2. Layer-by-Layer Responsibilities

### 2.1. Composition Root (`cmd/server/main.go`)

- **Single Responsibility**: Bootstrap and wire all application dependencies.
- Instantiates database connection pools, repositories, services, and HTTP handlers.
- Mounts routes and middleware chains.
- Spawns background workers (e.g., periodic expired token purger).
- Manages graceful shutdown via operating system signal trapping (`SIGINT`, `SIGTERM`).

### 2.2. HTTP Presentation Layer (`internal/handlers/`)

- **Single Responsibility**: Protocol mediation between HTTP and domain logic.
- Decodes JSON request bodies, URL query parameters, and route parameters (`chi.URLParam`).
- Triggers DTO validation via `input.Validate()`.
- Maps domain sentinel errors into semantic HTTP status codes (`400`, `401`, `403`, `404`, `409`, `422`, `500`).
- Emits standardized JSON responses using `utils.JSON()` and `utils.Error()`.
- Executes post-action side effects (such as invoking asynchronous file deletions on obsolete avatar/cover images).

### 2.3. Domain & Service Layer (`internal/service/`)

- **Single Responsibility**: Enforcement of business invariants, data transformation, and domain orchestration.
- Generates entity ULIDs (`utils.NewULID()`).
- Coordinates cryptographic operations (Argon2id password hashing, token generation).
- Orchestrates multi-step workflows (e.g., registering an account, creating an auth token, and dispatching a verification email via the mailer interface).
- Enforces system security rules, such as Refresh Token Rotation (RTR), reuse detection, and the **Super Admin Safeguard**.
- Emits strongly-typed sentinel errors (e.g., `ErrNotFound`, `ErrDuplicate`, `ErrUnauthorized`).

### 2.4. Data Access Layer (`internal/repository/`)

- **Single Responsibility**: Data persistence and retrieval abstraction.
- Exposes clean interfaces consumed by the service layer.
- Implements queries using **SQLC** generated structs (`internal/repository/sqlc`).
- Handles complex dynamic SQL queries with parameter binding (e.g., administrator user listings with multi-role filters and keyword search).
- Translates driver-specific database errors (e.g., MySQL error code 1062 for unique constraint violations) into standardized repository errors (`ErrDuplicate`).

---

## 3. Error Handling Architecture

The application categorizes errors into two distinct pipelines:

```text
                    Error Encountered
                            │
            ┌───────────────┴───────────────┐
            ▼                               ▼
 [ Validation Errors ]             [ Domain Errors ]
 (Multi-Input Validation)        (Service Sentinel Errors)
            │                               │
            ▼                               ▼
 utils.ValidationError(w, err)        errors.Is(err, target)
            │                               │
            ▼                               ▼
  HTTP 422 Unprocessable Entity      Semantic HTTP Status:
  {                                  ├── 401 Unauthorized
    "error": "validation failed",    ├── 403 Forbidden
    "errors": { ... }                ├── 404 Not Found
  }                                  └── 409 Conflict
```

### 3.1. Domain Sentinel Errors

Services define exported package-level sentinel errors:

```go
var (
    ErrUserNotFound           = errors.New("user not found")
    ErrUsernameExists         = errors.New("username already exists")
    ErrCannotDeleteLastSuperAdmin = errors.New("cannot delete the last active super admin")
)
```

Handlers evaluate errors using standard Go `errors.Is`:

```go
if errors.Is(err, service.ErrUserNotFound) {
    utils.Error(w, http.StatusNotFound, err.Error())
    return
}
if errors.Is(err, service.ErrCannotDeleteLastSuperAdmin) {
    utils.Error(w, http.StatusForbidden, err.Error())
    return
}
```

### 3.2. Multi-Input Field Validation

Instead of failing immediately on the first invalid field, every input DTO evaluates all constraints and compiles them into a `ValidationErrors` map:

```go
type ValidationErrors map[string]string
```

When errors exist, `utils.ValidationError(w, err)` formats the response into a structured JSON payload:

```json
{
  "error": "validation failed",
  "errors": {
    "username": "username must be between 3 and 30 characters",
    "email": "invalid email format"
  }
}
```

---

## 4. Entity Identity: Why ULID?

All primary keys and relational foreign keys use **ULID (Universally Unique Lexicographically Sortable Identifier)** stored as `CHAR(26)`.

| Metric                         | Sequential Integer (Auto-Increment) | Random UUID (UUIDv4)      | ULID (Adopted)                         |
| :----------------------------- | :---------------------------------- | :------------------------ | :------------------------------------- |
| **URL Enumeration Security**   | Vulnerable to scraping              | Secure                    | Secure (80-bit random tail)            |
| **B-Tree Index Fragmentation** | Optimal (Monotonic append)          | Severe random page splits | Optimal (Monotonic 48-bit time prefix) |
| **Decentralized Generation**   | Impossible (Requires DB lock)       | Supported                 | Supported (Client/App-side)            |
| **Sorting**                    | Numeric                             | Unordered                 | Chronological (`ORDER BY id DESC`)     |
| **JavaScript Client Safety**   | Truncates over $2^{53} - 1$         | String safe               | String safe (`CHAR(26)`)               |

---

## 5. Database Connection Pooling Best Practices

The database connection is managed via `internal/database/database.go` with enterprise production configurations:

```go
// Connection Pool tuning
db.SetMaxIdleConns(10)          // Retains idle connections ready for instant reuse
db.SetMaxOpenConns(100)         // Caps total concurrent connections to prevent DB exhaustion
db.SetConnMaxLifetime(time.Hour)// Refreshes stale connections gracefully
```

All migrations are run sequentially on startup using `golang-migrate` embedded into the Go binary (`//go:embed`), guaranteeing zero schema drift across environments.
