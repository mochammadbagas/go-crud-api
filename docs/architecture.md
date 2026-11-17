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
- Registers deep health and readiness probes (`/health`, `/health/live`, `/health/ready`) integrated with database connectivity pinging.
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
   HTTP 400 Bad Request               Semantic HTTP Status:
   {                                  ├── 401 Unauthorized
     "code": "VALIDATION_FAILED",     ├── 403 Forbidden
     "error": "validation failed",    ├── 404 Not Found
     "message": "validation failed",  └── 409 Conflict
     "errors": { ... }                (All output unified JSON)
   }
```

### 3.1. Unified Error Envelope & Domain Sentinel Errors

Services define exported package-level sentinel errors:

```go
var (
    ErrUserNotFound               = errors.New("user not found")
    ErrUsernameExists             = errors.New("username already exists")
    ErrCannotDeleteLastSuperAdmin = errors.New("cannot delete the last active super admin")
)
```

Handlers evaluate errors using standard Go `errors.Is` and return unified, structured error envelopes:

```go
if errors.Is(err, service.ErrUserNotFound) {
    utils.ErrorCode(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
    return
}
if errors.Is(err, service.ErrCannotDeleteLastSuperAdmin) {
    utils.ErrorCode(w, http.StatusForbidden, "CANNOT_DELETE_LAST_SUPER_ADMIN", err.Error())
    return
}
```

Every application error returns a production-grade envelope with a machine-readable `code` alongside human-readable `error` and `message` properties:

```json
{
  "code": "EMAIL_ALREADY_EXISTS",
  "error": "email is already registered",
  "message": "email is already registered"
}
```

If an endpoint does not specify an explicit code, `utils.Error(w, status, message)` automatically maps the HTTP status to a standard error code (`BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `RATE_LIMIT_EXCEEDED`, `INTERNAL_SERVER_ERROR`), guaranteeing that 100% of error responses carry a machine-readable code.

Global Chi router 404 (Not Found) and 405 (Method Not Allowed) handlers are configured to emit identical JSON envelopes instead of plain-text responses:

```json
{
  "code": "NOT_FOUND",
  "error": "endpoint not found",
  "message": "endpoint not found"
}
```

#### Application-Level Error Codes Dictionary

| Code | HTTP Status | Description / Trigger Scenario |
| :--- | :--- | :--- |
| `TOKEN_REQUIRED` | 401 Unauthorized | Missing or malformed `Authorization: Bearer <token>` header |
| `INVALID_OR_EXPIRED_TOKEN` | 401 Unauthorized | JWT access token is invalid, signature mismatch, or expired |
| `INVALID_CREDENTIALS` | 401 Unauthorized | Incorrect email or password during login |
| `TOKEN_EXPIRED` | 401 Unauthorized | Refresh token expired during token rotation |
| `TOKEN_REUSED` | 401 Unauthorized | Refresh token reuse detected (session hijacking mitigation) |
| `UNAUTHORIZED` | 401 Unauthorized | Generic authentication failure fallback |
| `INSUFFICIENT_PERMISSIONS` | 403 Forbidden | Authenticated user lacks the required RBAC role or permission |
| `ACCOUNT_INACTIVE` | 403 Forbidden | Account status is `inactive` or `suspended` |
| `CANNOT_DELETE_LAST_SUPER_ADMIN` | 403 Forbidden | Invariant safeguard preventing deletion of the last Super Admin |
| `CANNOT_DEACTIVATE_LAST_SUPER_ADMIN` | 403 Forbidden | Safeguard preventing deactivation of the last Super Admin |
| `CANNOT_DEMOTE_LAST_SUPER_ADMIN` | 403 Forbidden | Safeguard preventing revoking Super Admin role from the last holder |
| `CANNOT_DELETE_SYSTEM_ROLE` | 403 Forbidden | Protected system roles (`super_admin`, `admin`, `user`) cannot be deleted |
| `NOT_FOUND` | 404 Not Found | Route endpoint not found or generic record missing |
| `USER_NOT_FOUND` | 404 Not Found | Requested user ID does not exist in the database |
| `CATEGORY_NOT_FOUND` | 404 Not Found | Requested category ID or slug does not exist |
| `POST_NOT_FOUND` | 404 Not Found | Requested post ID or slug does not exist / not published |
| `ROLE_NOT_FOUND` | 404 Not Found | Requested role ID does not exist |
| `PERMISSION_NOT_FOUND` | 404 Not Found | Requested permission ID does not exist |
| `CONFLICT` | 409 Conflict | Generic uniqueness constraint violation |
| `EMAIL_ALREADY_EXISTS` | 409 Conflict | User registration or email update with an existing email |
| `USERNAME_ALREADY_EXISTS` | 409 Conflict | User registration or username update with an existing username |
| `CATEGORY_SLUG_EXISTS` | 409 Conflict | Category creation or update with an existing URL slug |
| `POST_SLUG_EXISTS` | 409 Conflict | Post creation or update with an existing URL slug |
| `ROLE_NAME_EXISTS` | 409 Conflict | Role creation with an existing system name |
| `PERMISSION_NAME_EXISTS` | 409 Conflict | Permission creation with an existing resource-action name |
| `EMAIL_ALREADY_VERIFIED` | 409 Conflict | Resending verification for an already verified email |
| `BAD_REQUEST` | 400 Bad Request | Malformed JSON body or invalid route parameter |
| `RATE_LIMIT_EXCEEDED` | 429 Too Many Requests | IP rate limit exceeded (e.g. max 10 auth requests/min) |
| `METHOD_NOT_ALLOWED` | 405 Method Not Allowed | HTTP method is not allowed for the requested endpoint |
| `INTERNAL_SERVER_ERROR` | 500 Internal Server Error | Unhandled server error or unexpected infrastructure failure |

### 3.2. Multi-Input Field Validation

Instead of failing immediately on the first invalid field, every input DTO evaluates all constraints and compiles them into a structured `ValidationErrors` map containing machine-readable error codes and human-readable messages:

```go
type FieldError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ValidationErrors map[string]FieldError
```

When errors exist, `utils.ValidationError(w, err)` formats the response into a structured JSON payload:

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
    }
  }
}
```

#### Field-Level Validation Error Codes Dictionary

Exported constants in `internal/utils/validation.go`:

| Code Constant | Machine Code (`code`) | Meaning / Purpose | Example Triggers |
| :--- | :--- | :--- | :--- |
| `ErrCodeRequired` | `REQUIRED` | Mandatory field is absent, empty, or whitespace-only | `full_name is required`, `email is required` |
| `ErrCodeTooShort` | `TOO_SHORT` | Input string length is below the minimum allowed bound | `password must be at least 8 characters long` |
| `ErrCodeTooLong` | `TOO_LONG` | Input string length exceeds the maximum allowed bound (e.g., 128 chars for passwords to prevent Argon2id CPU exhaustion DoS) | `password must not exceed 128 characters`, `bio must not exceed 500 characters` |
| `ErrCodeInvalidFormat` | `INVALID_FORMAT` | String fails regular expression or character set requirements | `username can only contain alphanumeric, underscore, hyphen`, `slug must be url-safe` |
| `ErrCodeInvalidEmail` | `INVALID_EMAIL` | Email fails RFC 5321 length rules (max 254 chars, local-part max 64 chars) or lacks a valid TLD (min 2 alphabetic chars) | `user@domain`, `user@localhost`, `user..test@example.com` |
| `ErrCodeReserved` | `RESERVED` | Input uses protected system identifiers | `admin`, `root`, `system`, `api`, `support`, `superuser` |
| `ErrCodeInvalidID` | `INVALID_ID` | Foreign key or relational ID is not a syntactically valid 26-character Crockford Base32 ULID | `category_id must be a valid 26-character ULID`, `permission_ids must contain valid ULIDs` |
| `ErrCodeInvalidStatus` | `INVALID_STATUS` | Value does not belong to the allowed enumeration set for status | `status must be one of: draft, published` |
| `ErrCodeInvalidValue` | `INVALID_VALUE` | Relational value contradiction or disallowed payload combination | `new password cannot be the same as old password`, non-negative numeric constraints |

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
db.SetMaxIdleConns(10)               // Retains idle connections ready for instant reuse
db.SetMaxOpenConns(100)              // Caps total concurrent connections to prevent DB exhaustion
db.SetConnMaxLifetime(time.Hour)     // Refreshes stale connections gracefully
db.SetConnMaxIdleTime(10 * time.Minute) // Closes idle connections before remote dropouts
```

All migrations are run sequentially on startup using `golang-migrate` embedded into the Go binary (`//go:embed`), guaranteeing zero schema drift across environments.
