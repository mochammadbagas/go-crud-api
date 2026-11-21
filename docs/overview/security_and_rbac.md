# Security Architecture & Role-Based Access Control (RBAC)

**Document Type:** Security Specification & Authorization Guide  
**Target Audience:** Backend Engineers, Security Auditors, and Frontend Developers  
**Standards:** RFC 6749 (OAuth 2.0), RFC 7519 (JWT), RFC 9106 (Argon2id), OWASP ASVS 4.0

---

## 1. Authentication Engine: JWT + Refresh Token Rotation (RTR)

The platform implements a modern dual-token session architecture that balances high-performance stateless verification with cryptographic session revocation.

```text
[ Login Request ] ────► Verify Argon2id Password
                             │
                             ▼
     ┌───────────────────────┴───────────────────────┐
     ▼                                               ▼
[ Access Token ]                               [ Refresh Token ]
- Stateless HS256 JWT                          - Cryptographic random 64-char string
- Lifetime: 15 minutes                         - Lifetime: 7 days
- Contains: user_id, email, roles              - Stored as SHA-256 hash in database
- Delivered in JSON Body                       - Delivered via HttpOnly Cookie + JSON
```

### 1.1. Refresh Token Rotation (RTR) & Reuse Detection

Whenever a client requests a new access token via `POST /api/v1/auth/refresh`, the refresh token itself is rotated:

```text
[ Step 1: Normal Refresh ]
Client sends Refresh Token A (Family X)
  ├── 1. Verify Token A is active and unexpired
  ├── 2. Mark Token A as revoked (is_revoked = TRUE)
  ├── 3. Issue new Refresh Token B (same Family X)
  └── 4. Issue new JWT Access Token

[ Step 2: Adversary Token Interception / Replay ]
Adversary presents old Refresh Token A (already revoked!)
  ├── 1. Database detects: Token A is revoked
  ├── 2. SYSTEM TRIGGERS REUSE DETECTION ALERT:
  │      "Refresh token reuse detected! Revoking full token family"
  ├── 3. Immediately revokes ALL tokens belonging to Family X!
  └── 4. Rejects request with HTTP 401 Unauthorized
```

This guarantees that if an attacker intercepts a token from transit or storage, any subsequent attempt by either the legitimate user or the attacker immediately kills the entire session.

### 1.2. Hybrid Delivery Strategy (Web vs Mobile/API)

The API supports both browser clients and native/mobile applications:

| Client Type                         | Refresh Token Delivery                                 | Request Extraction                    | Security Benefit                                |
| :---------------------------------- | :----------------------------------------------------- | :------------------------------------ | :---------------------------------------------- |
| **Web / SPA (React, Vue, Next.js)** | `HttpOnly`, `SameSite=Lax`, `Path=/api/v1/auth` cookie | Automatic browser cookie transmission | Immune to XSS token theft via client JavaScript |
| **Mobile / Postman / CLI**          | Returned in JSON body (`refresh_token`)                | `X-Refresh-Token` header or JSON body | Works without cookie jars or webview storage    |

### 1.3. Automated Multi-Device Session Revocation (`RevokeAllForUser`)

To minimize window-of-vulnerability after account state mutations, the system invokes `tokenRepo.RevokeAllForUser` to immediately invalidate all active refresh tokens across all devices upon:

1. **Password Mutation**: Manual password changes or successful password resets.
2. **Email Change**: Confirmation of a new email address.
3. **Account Deactivation or Suspension**: Immediate session severance when an administrator alters account status (`inactive`, `suspended`).
4. **Role Modification / Demotion**: Modifying roles via `AssignUserRoles` invalidates existing token families to enforce immediate re-authentication with new privileges.
5. **Account Deletion**: Complete deletion of user records cleans up all token lineages.

---

## 2. Password Hashing & Account Recovery

### 2.1. Argon2id Password Hashing

Passwords are never hashed using outdated algorithms (MD5, SHA, or standard bcrypt). The system utilizes **Argon2id** (RFC 9106), the winner of the Password Hashing Competition:

- **Algorithm**: Argon2id
- **Memory**: 64 MB (`64 * 1024` KiB)
- **Iterations**: 1 pass
- **Parallelism**: 2 threads
- **Salt Length**: 16 bytes cryptographically random entropy

### 2.2. Ephemeral One-Time Tokens (SHA-256)

For email verification, password reset, and email change confirmation:

1. The server generates 32 bytes of secure entropy (`crypto/rand`).
2. The raw 64-character hex token is sent to the user's email address.
3. The database stores only the one-way hash: `SHA-256(rawToken)`.
4. Even in the event of an unauthorized database read, pending reset tokens cannot be compromised.

### 2.3. Anti-Enumeration Protection

Endpoints that accept an email address (`/api/v1/auth/forgot-password` and `/api/v1/auth/resend-verification`) strictly return **HTTP 200 OK** regardless of whether the email exists in the database. This prevents attackers from enumerating valid registered user accounts.

---

## 3. Granular Role-Based Access Control (RBAC)

The authorization subsystem decouples user identities from permissions via a flexible relational multi-role model.

```text
[ users ] 1 ──── ∞ [ user_roles ] ∞ ──── 1 [ roles ]
                                              1
                                              │
                                              ∞
                                     [ role_permissions ]
                                              ∞
                                              │
                                              1
                                       [ permissions ]
```

### 3.1. Permission Syntax: `resource.action`

Permissions follow a standardized dot-notated syntax:

- `users.read`, `users.create`, `users.update`, `users.delete`
- `roles.read`, `roles.create`, `roles.update`, `roles.delete`
- `permissions.read`, `permissions.create`, `permissions.update`, `permissions.delete`
- `articles.read`, `articles.create`, `articles.update`, `articles.delete`
- `categories.manage`
- `tags.read`, `tags.manage`
- `media.upload`, `media.read`, `media.delete`
- `analytics.read`
- `settings.manage`

### 3.2. Authorization Middleware Pipeline

Routes are protected using composable Chi middleware decorators:

```go
// 1. Verify valid JWT bearer token
r.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

// 2. Protect route by Role
r.With(middleware.RequireRole("admin")).Get("/admin/reports", handler.GetReports)

// 3. Protect route by Granular Permission
r.With(middleware.RequirePermission("users.delete", userRepo)).Delete("/admin/users/{id}", handler.DeleteUser)
```

### 3.3. Super Admin Wildcard Bypass (`*`)

Accounts possessing the `super_admin` role automatically bypass granular permission lookups. The `PermissionMiddleware` detects the `super_admin` role claim in the JWT and allows the request through without querying the database.

### 3.4. In-Memory Permission Caching with Automatic Invalidation

To eliminate redundant database lookups on high-frequency admin endpoints, `middleware.RequirePermission` caches user permissions in an in-memory thread-safe `sync.Map` with a 1-minute TTL. Whenever an administrator modifies role assignments (`AssignUserRoles`) or permission bindings (`AssignPermissionsToRole`), `middleware.ClearUserPermissionCache` is called immediately, ensuring authorization changes take effect instantly across the system.

### 3.5. Dual-Layer Role Synchronization (`users.role` & `user_roles`)

The architecture maintains dual-layer role persistence:

1. **Relational RBAC (`user_roles`)**: Supports many-to-many role mappings per user.
2. **Denormalized Legacy Column (`users.role`)**: Holds the primary user role (`super_admin` > `admin` > `user`) for high-speed JWT claim generation and backward compatibility.

Whenever roles are updated via `userRepo.SetRoles`, the transaction atomically synchronizes both the pivot table and the primary `users.role` column, preventing role claim desynchronization.

---

## 4. Administrative Safeguards: Super Admin Safeguard

To prevent accidental self-lockout or total loss of administrative access, domain services enforce the **Super Admin Safeguard**:

```text
IF target user has 'super_admin' role
AND count(active super_admin users) <= 1
AND attempted action is:
    ├── Delete user account
    ├── Change status to 'inactive' or 'suspended'
    └── Remove 'super_admin' role from user
THEN
    BLOCK OPERATION IMMEDIATELY!
    Return HTTP 403 Forbidden: "cannot modify or delete the last active super admin"
```

This invariant guarantees that a system can never be rendered administrator-less through API calls.

---

## 5. Injection Defense & File Security

### 5.1. SQL Injection Prevention via SQLC

The application avoids raw string concatenation in SQL queries. All database operations are compiled via **SQLC** into prepared statements with parameterized inputs (`?`). Even dynamic administrative filtering queries use parameterized argument binding, neutralizing SQL injection vectors.

### 5.2. Cross-Site Scripting (XSS) Sanitization

Content modules supporting rich text HTML (e.g. articles) utilize the **Bluemonday** sanitization library configured with the strict User-Generated Content (`UGCPolicy()`). Raw `<script>` tags, inline dangerous event handlers (`onload`, `onclick`, `onerror`), and unsafe URI pseudo-protocols (`javascript:`, `data:`) are automatically stripped before HTML persistence into database `LONGTEXT` columns.

### 5.3. Storage Security & Path Traversal Guards

- **Magic Bytes Sniffing**: Rather than trusting client-provided `Content-Type` headers or file extensions, uploads are inspected via `http.DetectContentType` to verify binary image headers (`image/jpeg`, `image/png`, `image/webp`, `image/gif`).
- **Zero-Leakage ULID Renaming**: Client-supplied filenames (e.g., `../../etc/passwd` or `shell.php`) are discarded entirely. The file is assigned a randomized ULID (`<ULID>.<ext>`), eliminating path traversal vulnerabilities.
- **Directory Browsing Prevention**: The local static file server wraps the filesystem with a restricted filesystem (`noDirListingFS`) returning HTTP 404 on directory paths without `index.html`, eliminating unauthorized asset discovery.
- **Automated Orphan Cleanup & In-Use Protection**: Mutating an avatar automatically purges the obsolete file from storage. For media library assets, database relations in `articles` and `article_media` enforce referential safety, preventing active media from accidental deletion.

---

## 6. HTTP Hardening & Denial of Service (DoS) Defense

### 6.1. Request Body Size Limiting (`http.MaxBytesReader`)

All incoming HTTP request bodies are strictly bounded using `http.MaxBytesReader` to prevent memory exhaustion and disk-filling Denial of Service (DoS) attacks:

- **JSON Request Payloads**: Enforces a hard **1 MB limit** (`1 << 20`) via `utils.DecodeJSON`. Payload overflows return HTTP 413 `PAYLOAD_TOO_LARGE`.
- **User Avatar Uploads**: Enforces a hard **5 MB limit** (`5 << 20`) on multipart streams before parsing in `user_handler.go`.
- **Media Library Uploads**: Enforces a hard **10 MB limit** (`10 << 20`) on multipart streams in `media_handler.go`.

### 6.2. OWASP Security Headers Middleware

All HTTP responses carry standard OWASP security headers injected via `middleware.SecurityHeaders()`:

- `X-Content-Type-Options: nosniff`: Prevents MIME type sniffing.
- `X-Frame-Options: DENY`: Prevents clickjacking by forbidding iframe framing.
- `Referrer-Policy: strict-origin-when-cross-origin`: Restricts referrer leaks.
- `X-XSS-Protection: 1; mode=block`: Activates browser reflected XSS filters.
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`: Rejects framing and unauthorized execution.
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`: Injected automatically whenever traffic is served over HTTPS/TLS or forwarded via `X-Forwarded-Proto: https` in production, enforcing transport-layer encryption.

### 6.3. Strict Route Parameter & ULID Validation

All handlers serving parameterized routes (`/users/{id}`, `/admin/articles/{id}`, `/admin/roles/{id}`, `/admin/permissions/{id}`, `/admin/media/{id}`) validate the input format using `utils.IsValidULID`. Malformed or non-Crockford Base32 parameters are rejected immediately with HTTP 400 `INVALID_ID`, shielding database queries from executing against malformed identifiers.
