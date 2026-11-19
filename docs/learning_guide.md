# Go REST API Starter Kit — Developer Learning Guide

**Document Type:** Educational Architecture Walkthrough & Idiomatic Go Guide  
**Target Audience:** Software Engineers, Backend Developers, and System Architects  
**Prerequisites:** Basic understanding of Go syntax (structs, interfaces, goroutines)  
**Standard:** Clean Architecture, Idiomatic Go Patterns

---

## 1. Core Go Concepts Applied in This Project

This guide explains the specific Go language idioms, concurrency patterns, and design principles implemented throughout the **Go REST API Starter Kit (`go-crud-api`)**.

| Concept                                  | Implementation Location                                            | Architectural Purpose & Idiomatic Benefit                                                                                                                                                                                                                                                                             |
| :--------------------------------------- | :----------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Decentralized ULID Identity**          | `internal/utils/ulid.go`                                           | Uses 26-character Crockford Base32 strings (`CHAR(26)`) via `github.com/oklog/ulid/v2`. Prevents ID enumeration, ensures chronological sorting via millisecond timestamp prefixes, and prevents 53-bit JavaScript integer truncation.                                                                                 |
| **Pluggable Adapter Pattern**            | `internal/mailer/`, `internal/storage/`                            | Isolates external infrastructure behind clean Go interfaces (`Mailer` and `Storage`). Enables zero-config local development (`LogMailer`, `LocalStorage`) and seamless production switching (`ResendMailer`, Cloudflare R2 / AWS S3) via configuration flags.                                                         |
| **Struct Pointers for Nullable Types**   | `internal/models/`                                                 | Uses pointers (`*string`, `*time.Time`) to represent nullable database fields in Go structs, mapping cleanly to SQL `NULL` and JSON `null` / `omitempty`.                                                                                                                                                             |
| **Multi-Input Field Validation**         | `internal/utils/validation.go`, `internal/models/`                 | Replaces fail-fast early returns with a unified `ValidationErrors map[string]FieldError` collector implementing Go's standard `error` interface. Evaluates all field constraints simultaneously and returns structured errors with machine-readable `code` and human-readable `message` for client UI forms and i18n. |
| **Dependency Inversion Principle (DIP)** | `internal/service/`, `internal/repository/`                        | Higher-level business services depend exclusively upon repository and infrastructure interfaces, enabling full modularity, loose coupling, and straightforward unit testing via in-memory mock repositories.                                                                                                          |
| **Compile-Time Type-Safe SQL**           | `db/queries/`, `internal/repository/sqlc/`                         | Uses `sqlc` to compile raw SQL queries directly into type-safe Go code. Delivers native database performance and query visibility without the abstraction overhead or runtime reflection of traditional ORMs.                                                                                                         |
| **Embedded Binary Assets**               | `db/embed.go`                                                      | Embeds migration DDL scripts directly into the compiled executable binary using Go's `//go:embed` directive, eliminating external migration script dependencies during deployment.                                                                                                                                    |
| **Concurrency & Synchronization**        | `cmd/server/main.go`                                               | Uses goroutines and channels to coordinate the HTTP server listener, periodic background token cleanup workers (`time.Ticker`), and OS termination signal interception (`os.Signal`).                                                                                                                                 |
| **Sentinel Error Architecture**          | `internal/repository/`, `internal/service/`                        | Defines domain sentinel errors (`ErrNotFound`, `ErrDuplicate`, `ErrTokenReused`, `ErrUnauthorized`) compared using `errors.Is` across layer boundaries and translated into unified JSON error envelopes with machine-readable `code` and human-readable `message`.                                                    |
| **Higher-Order Middleware Functions**    | `internal/middleware/auth.go`, `internal/routes/routes.go`         | Implements standard `func(http.Handler) http.Handler` middleware wrappers on the Chi router to enforce rate limiting, authentication, CORS, and RBAC authorization pipelines.                                                                                                                                         |
| **Context Propagation & Timeouts**       | `cmd/server/main.go`, `internal/middleware/auth.go`                | Leverages `context.Context` for request lifecycle management: `context.WithTimeout` for graceful shutdown, `context.WithCancel` for worker termination, and `context.WithValue` for propagating authenticated identity claims.                                                                                        |
| **Comprehensive Test Automation**        | `internal/handlers/handlers_test.go`, `internal/service/*_test.go` | Utilizes table-driven test patterns, in-memory mock repositories, and `net/http/httptest` to verify HTTP responses, status codes, and domain invariants without requiring a live database.                                                                                                                            |

---

## 2. Idiomatic Go Deep Dives

### 2.1. Pointer vs Value Semantics for Nullable SQL Types

In relational databases, columns such as `bio`, `phone_number`, and `avatar_url` are often nullable (`NULL`). Go primitives (`string`, `int`) do not have a `null` state—they default to their zero values (`""`, `0`).

Instead of using verbose `sql.NullString` in domain models, this project uses standard Go pointers:

```go
type UserProfile struct {
    UserID      string    `json:"user_id"`
    FullName    string    `json:"full_name"`
    Bio         *string   `json:"bio,omitempty"`          // nil represents SQL NULL
    PhoneNumber *string   `json:"phone_number,omitempty"` // nil represents SQL NULL
    AvatarURL   *string   `json:"avatar_url,omitempty"`   // nil represents SQL NULL
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Benefits**:

- Clean JSON serialization: If `Bio == nil`, `json:"bio,omitempty"` omits the key or serializes as `null`.
- Zero database coupling: Domain models remain pure Go structs without importing `database/sql`.
- Conversion to `sql.NullString` happens strictly at the repository boundary (`internal/repository/`).

---

### 2.2. Dependency Inversion with Go Interfaces

Go interfaces are **satisfied implicitly**. A struct does not declare `implements Repository`; it simply implements the required method signatures.

```go
// 1. Service defines what it needs (the contract)
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*models.User, error)
    UpdatePassword(ctx context.Context, id string, hash string) error
}

// 2. Service receives the interface via constructor
type AuthService struct {
    userRepo UserRepository
}

func NewAuthService(userRepo UserRepository, ...) *AuthService {
    return &AuthService{userRepo: userRepo}
}
```

**Testing Advantage**:
In unit tests (`internal/service/auth_service_test.go`), we do not need a live MySQL connection. We pass an in-memory struct implementing `UserRepository`, allowing instantaneous, deterministic test execution.

---

### 2.3. Background Workers with Goroutines & Tickers

In `cmd/server/main.go`, expired tokens are cleaned up periodically using a background goroutine, non-blocking channels, and `time.Ticker`:

```go
func startTokenCleanup(ctx context.Context, tokenRepo repository.RefreshTokenRepository, authTokenRepo repository.AuthTokenRepository, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    // 1. Initial cleanup upon server startup
    _ = tokenRepo.DeleteExpired(ctx)
    _ = authTokenRepo.DeleteExpiredOrUsed(ctx)

    // 2. Loop continuously until context cancellation
    for {
        select {
        case <-ctx.Done():
            return // Server shutting down; exit goroutine cleanly
        case <-ticker.C:
            _ = tokenRepo.DeleteExpired(ctx)
            _ = authTokenRepo.DeleteExpiredOrUsed(ctx)
        }
    }
}
```

**Why this is idiomatic**:

- The `select` statement prevents busy-waiting and consumes zero CPU cycles while idle.
- Passing `context.Context` allows graceful worker termination when the server shuts down.
- `defer ticker.Stop()` releases the underlying runtime timer resource.

---

### 2.4. Graceful Shutdown with OS Signal Trapping

When deploying in Docker or Kubernetes, a container receives a `SIGTERM` or `SIGINT` signal before termination. Abruptly killing the process drops active HTTP requests.

The server implements graceful shutdown:

```go
// 1. Create buffered signal channel
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

// 2. Block until signal received
<-quit
slog.Info("Shutdown signal received. Shutting down server gracefully...")

// 3. Create 10-second drain window
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 4. Shutdown gracefully (stops accepting new connections, waits for active handlers)
if err := srv.Shutdown(ctx); err != nil {
    slog.Error("Graceful server shutdown failed, forcing exit", slog.Any("error", err))
    os.Exit(1)
}
slog.Info("Server stopped safely.")
```

---

### 2.5. Higher-Order Functions for HTTP Middleware

Chi middlewares in Go are higher-order functions adhering to the `func(http.Handler) http.Handler` signature:

```go
func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tokenString := extractTokenFromHeader(r)
            claims, err := utils.ValidateJWT(tokenString, jwtSecret)
            if err != nil {
                utils.Error(w, http.StatusUnauthorized, "unauthorized")
                return
            }

            // Propagate authenticated identity downstream via Context
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

### 2.6. Table-Driven Unit Testing with `httptest`

The test suite leverages Go's standard table-driven test pattern combined with `net/http/httptest` to verify HTTP endpoints:

```go
func TestFieldLevelValidationResponse(t *testing.T) {
    tests := []struct {
        name         string
        payload      string
        expectedCode int
        expectedKeys []string
    }{
        {
            name:         "Empty username, invalid email, short password",
            payload:      `{"username": "", "email": "invalid-format", "password": "123"}`,
            expectedCode: http.StatusUnprocessableEntity,
            expectedKeys: []string{"username", "email", "password"},
        },
        {
            name:         "Missing email only",
            payload:      `{"username": "validuser", "email": "", "password": "password123"}`,
            expectedCode: http.StatusUnprocessableEntity,
            expectedKeys: []string{"email"},
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tc.payload))
            req.Header.Set("Content-Type", "application/json")
            rr := httptest.NewRecorder()

            router.ServeHTTP(rr, req)

            if rr.Code != tc.expectedCode {
                t.Fatalf("expected status %d, got %d", tc.expectedCode, rr.Code)
            }
        })
    }
}
```

---

## 3. Related Documentation References

- For the complete system architecture, clean layering, and error handling rules, see [docs/architecture.md](architecture.md).
- For step-by-step instructions on implementing a new CRUD feature, see [docs/how_to_add_new_module.md](how_to_add_new_module.md).
- For details on the security architecture, JWT rotation, and RBAC safeguards, see [docs/security_and_rbac.md](security_and_rbac.md).
- For deployment guidelines, Docker Compose, and cloud configuration, see [docs/deployment.md](deployment.md).
