# Response Envelope & Standard Error Codes

**Document Type:** API Specification & Client Error Handling Reference  
**Target Audience:** Frontend Developers, Mobile Engineers, QA Automation, and Integration Partners  
**Status:** Production Standard

---

## 1. Unified Response Envelope Standard

All API endpoints return JSON responses with standardized envelope schemas.

### 1.1 Single Resource Response

Successful operations returning a single object wrap the payload inside the `data` field:

```json
{
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "username": "johndoe",
    "email": "john@example.com",
    "role": "admin",
    "created_at": "2026-11-20T10:00:00Z",
    "updated_at": "2026-11-20T10:00:00Z"
  }
}
```

### 1.2 Paginated Resource Collection

Collections with pagination wrap the array in `data` and append metadata in `meta`:

```json
{
  "data": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "title": "Scaling Go REST APIs",
      "slug": "scaling-go-rest-apis",
      "status": "published"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 142,
    "total_pages": 8
  }
}
```

### 1.3 Action Confirmation / Empty Success

Operations such as deletion or verification confirmation that do not return an entity return a structured confirmation message:

```json
{
  "message": "article deleted successfully"
}
```

---

## 2. Standard Error Envelope

Every error response across domain validation, authentication, authorization, and unexpected failures follows a uniform contract:

```json
{
  "code": "STRING_IDENTIFIER",
  "error": "human-readable error description",
  "message": "human-readable error description",
  "errors": {}
}
```

### Multi-Field Validation Error Schema (HTTP 400)

When payload validation fails (`input.Validate()`), the response aggregates all validation failures without early return:

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

---

## 3. Master Error Codes Directory

Frontend applications should switch on `code` rather than parsing error message strings.

### 3.1 Authentication & Authorization (`/api/v1/auth`, RBAC)

| HTTP Status        | Error `code`                     | Description / Cause                                                                    | Recommended Client Action                               |
| :----------------- | :------------------------------- | :------------------------------------------------------------------------------------- | :------------------------------------------------------ |
| `401 Unauthorized` | `UNAUTHORIZED`                   | Missing, malformed, or expired JWT Access Token.                                       | Attempt refresh via `/api/v1/auth/refresh`.             |
| `401 Unauthorized` | `INVALID_CREDENTIALS`            | Incorrect email or password during login.                                              | Prompt user to verify email/password.                   |
| `401 Unauthorized` | `TOKEN_EXPIRED`                  | Refresh token or email action token has expired.                                       | Require fresh user login.                               |
| `401 Unauthorized` | `TOKEN_REVOKED`                  | Refresh token was manually revoked.                                                    | Clear local tokens and redirect to login.               |
| `401 Unauthorized` | `TOKEN_REUSED`                   | **Security Alert**: Intercepted/revoked token was reused. All session families killed. | Clear all tokens, display security notice, force login. |
| `401 Unauthorized` | `EMAIL_NOT_VERIFIED`             | Account has not verified email address.                                                | Display verification banner with "Resend email" action. |
| `403 Forbidden`    | `FORBIDDEN`                      | Missing required role or granular permission (`resource.action`).                      | Hide forbidden UI controls or display 403 screen.       |
| `403 Forbidden`    | `ACCOUNT_INACTIVE`               | Account status is `inactive` or `suspended`.                                           | Display account suspension message with support link.   |
| `403 Forbidden`    | `CANNOT_DELETE_LAST_SUPER_ADMIN` | Attempted to delete/demote the last active super admin.                                | Display invariant error modal.                          |

### 3.2 Request Validation & Route Errors

| HTTP Status              | Error `code`         | Description / Cause                                           | Recommended Client Action                           |
| :----------------------- | :------------------- | :------------------------------------------------------------ | :-------------------------------------------------- |
| `400 Bad Request`        | `VALIDATION_FAILED`  | One or more payload fields failed constraint checks.          | Map `errors[field].message` to form inputs.         |
| `400 Bad Request`        | `INVALID_ID`         | Route parameter is not a valid 26-char Crockford Base32 ULID. | Validate route parameter format before dispatching. |
| `400 Bad Request`        | `BAD_REQUEST`        | Malformed JSON or unparseable request body.                   | Inspect payload formatting.                         |
| `404 Not Found`          | `NOT_FOUND`          | Route or resource does not exist.                             | Display 404 page.                                   |
| `405 Method Not Allowed` | `METHOD_NOT_ALLOWED` | HTTP method not permitted on this route.                      | Verify HTTP verb (GET/POST/PUT/DELETE).             |

### 3.3 Domain & Resource Conflicts

| HTTP Status    | Error `code`              | Description / Cause                                           | Recommended Client Action                       |
| :------------- | :------------------------ | :------------------------------------------------------------ | :---------------------------------------------- |
| `409 Conflict` | `EMAIL_ALREADY_EXISTS`    | Email address is already registered.                          | Prompt user to login or use a different email.  |
| `409 Conflict` | `USERNAME_ALREADY_EXISTS` | Username is already taken.                                    | Prompt user to select an alternative username.  |
| `409 Conflict` | `SLUG_ALREADY_EXISTS`     | Article or category slug is already used.                     | Append an incremental suffix to the slug.       |
| `409 Conflict` | `MEDIA_IN_USE`            | Cannot delete media asset because it is embedded in articles. | Inform user which articles reference the asset. |
| `409 Conflict` | `CATEGORY_HAS_ARTICLES`   | Cannot delete category because articles are assigned to it.   | Prompt user to reassign articles first.         |

### 3.4 Infrastructure & System Health

| HTTP Status                 | Error `code`            | Description / Cause                                    | Recommended Client Action                       |
| :-------------------------- | :---------------------- | :----------------------------------------------------- | :---------------------------------------------- |
| `429 Too Many Requests`     | `RATE_LIMIT_EXCEEDED`   | Request limit reached (10/min auth, 60/min analytics). | Inspect `Retry-After` header and wait.          |
| `500 Internal Server Error` | `INTERNAL_SERVER_ERROR` | Unhandled server error (details masked for security).  | Display generic failure toast; contact support. |
| `503 Service Unavailable`   | `SERVICE_UNAVAILABLE`   | Database health probe ping failed or maintenance mode. | Retry with backoff.                             |
