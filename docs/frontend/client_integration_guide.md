# Client Integration Guide (Frontend & API Consumers)

**Document Type:** Client Integration & Protocol Specification  
**Target Audience:** Frontend Engineers (React, Next.js, Vue, Svelte), Mobile Developers (Flutter, React Native, Swift, Kotlin), and Third-Party API Consumers  
**Standard Protocols:** RFC 6749 (OAuth 2.0 / Refresh Token Rotation), RFC 7519 (JWT), RFC 7231 (HTTP Semantics)

---

## 1. Authentication & Session Architecture

The API uses a **Dual-Token Session Architecture** combining stateless, short-lived JSON Web Tokens (JWT) with stateful, rotating Refresh Tokens (RTR).

```text
[ Client (Browser / Mobile) ]
           │
           │ 1. POST /api/v1/auth/login { email, password }
           ▼
[ API Server ] ──► Verifies Argon2id Hash
           │
           ├──► Generates Access Token (JWT, 15 min lifetime)
           ├──► Generates Refresh Token (Cryptographic 64-char string, 7 days lifetime)
           │
           ▼
[ Client Response ]
  ├── Access Token  ──► Memory (React State, Zustand, Redux) or Secure Storage
  └── Refresh Token ──► HttpOnly Cookie (Browsers) OR JSON Body / X-Refresh-Token (Mobile)
```

### 1.1 Token Specifications

| Token Type        | Lifespan   | Format             | Client Storage Recommendation                                                | Purpose                                                                   |
| :---------------- | :--------- | :----------------- | :--------------------------------------------------------------------------- | :------------------------------------------------------------------------ |
| **Access Token**  | 15 Minutes | Signed HS256 JWT   | In-Memory (JavaScript variable/state)                                        | Sent via `Authorization: Bearer <token>` on protected routes.             |
| **Refresh Token** | 7 Days     | 64-char hex string | `HttpOnly` Cookie (Web) or OS Keychain / EncryptedSharedPreferences (Mobile) | Used solely to obtain a new Access Token via `POST /api/v1/auth/refresh`. |

### 1.2 Hybrid Delivery Strategy (Web vs Native Mobile)

The backend natively supports two delivery mechanisms without configuration changes:

1. **Web / SPA (Browsers)**:
   - When logging in or refreshing tokens, the server automatically sets a secure cookie:
     ```http
     Set-Cookie: refresh_token=<token>; Path=/api/v1/auth; HttpOnly; SameSite=Lax; Max-Age=604800
     ```
   - In production (`APP_ENV=production`), the `Secure` flag is enforced (requires HTTPS).
   - In client requests (`fetch` or `axios`), include `{ withCredentials: true }` or `credentials: 'include'`. The browser automatically transmits the cookie to `/api/v1/auth/refresh` and `/api/v1/auth/logout`.

2. **Mobile & CLI Clients**:
   - The login/refresh responses also return `refresh_token` in the JSON response body.
   - For refresh requests, mobile clients pass the token either in the JSON body:
     ```json
     { "refresh_token": "your_64_character_refresh_token" }
     ```
     or via the custom header:
     ```http
     X-Refresh-Token: your_64_character_refresh_token
     ```

### 1.3 Refresh Token Rotation (RTR) & Reuse Detection Guard

Every call to `POST /api/v1/auth/refresh` **consumes and revokes** the submitted refresh token and issues a **brand new** refresh token belonging to the same _Token Family_.

> [!CAUTION]
> **Automatic Family Revocation on Replay Attacks:**  
> If an expired or already-revoked refresh token is re-submitted (a classic indicator that the token was intercepted), the backend triggers **Reuse Detection**. All active tokens in that user's token family are immediately revoked with HTTP 401 `TOKEN_REUSED`. The legitimate user must log in again with credentials.

---

## 2. Production HTTP Interceptor Implementation

Below is the standard, production-ready implementation using **Axios**. It handles:

- Attaching the Bearer token to outgoing requests.
- Queuing concurrent requests while a token refresh is in progress (preventing multiple refresh calls).
- Preventing infinite 401 retry loops.
- Seamless fallback to logout when refresh fails.

### 2.1 Axios Implementation (TypeScript / JavaScript)

```typescript
import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true, // Crucial: enables HttpOnly cookie transmission for web
  headers: {
    "Content-Type": "application/json",
  },
});

// In-memory access token storage
let currentAccessToken: string | null = null;
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: any) => void;
}> = [];

export const setAccessToken = (token: string | null) => {
  currentAccessToken = token;
};

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach((promise) => {
    if (error) {
      promise.reject(error);
    } else if (token) {
      promise.resolve(token);
    }
  });
  failedQueue = [];
};

// 1. Request Interceptor: Attach Access Token
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (currentAccessToken && !config.headers.Authorization) {
      config.headers.Authorization = `Bearer ${currentAccessToken}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

// 2. Response Interceptor: Handle 401 & Automatic Refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // Bypass refresh logic if the request is already to the auth endpoint or already retried
    if (
      !error.response ||
      error.response.status !== 401 ||
      originalRequest._retry ||
      originalRequest.url?.includes("/api/v1/auth/login") ||
      originalRequest.url?.includes("/api/v1/auth/refresh")
    ) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      // If refresh is already in flight, queue this request
      return new Promise<string>((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      })
        .then((newToken) => {
          originalRequest.headers.Authorization = `Bearer ${newToken}`;
          return apiClient(originalRequest);
        })
        .catch((err) => Promise.reject(err));
    }

    originalRequest._retry = true;
    isRefreshing = true;

    try {
      // Call refresh endpoint (browser sends HttpOnly cookie automatically)
      const { data } = await axios.post(
        `${API_BASE_URL}/api/v1/auth/refresh`,
        {},
        { withCredentials: true },
      );

      const newAccessToken = data.data.access_token;
      setAccessToken(newAccessToken);
      processQueue(null, newAccessToken);

      originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
      return apiClient(originalRequest);
    } catch (refreshError) {
      processQueue(refreshError, null);
      setAccessToken(null);

      // Trigger redirect to login page or dispatch logout action
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("auth:session-expired"));
      }
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  },
);
```

---

## 3. Rate-Limiting & Backoff Strategy

The API enforces token-bucket rate limiting using IP address identifiers:

- **Authentication Routes (`/api/v1/auth/*`)**: Max **10 requests/minute**.
- **Public Analytics Ingestion (`/api/v1/articles/{id}/analytics`)**: Max **60 requests/minute**.

When rate limits are exceeded, the API responds with:

- **HTTP Status:** `429 Too Many Requests`
- **JSON Payload:**
  ```json
  {
    "code": "RATE_LIMIT_EXCEEDED",
    "error": "too many authentication attempts, please try again later",
    "message": "too many authentication attempts, please try again later"
  }
  ```
- **Response Headers:**
  - `X-RateLimit-Limit`: Maximum allowed requests per window.
  - `X-RateLimit-Remaining`: Remaining request quota.
  - `X-RateLimit-Reset`: Unix timestamp when the quota resets.
  - `Retry-After`: Recommended wait time in seconds before retrying.

### Client Backoff Recommendation

When handling HTTP 429, extract `Retry-After` (or default to exponential backoff `2^attempt * 1000ms`) and disable client-side submit buttons until the cooldown expires.

---

## 4. Media Upload Guidelines

The API provides two endpoints for file uploads:

1. **User Avatar (`POST /api/v1/users/avatar`)**: Max **5MB**, restricted to image formats.
2. **Centralized Media Library (`POST /api/v1/admin/media`)**: Max **10MB**, requires `media.upload` permission.

### 4.1 Upload Request Requirements

- Content-Type must be `multipart/form-data`.
- Field name: `file`.
- Allowed MIME types: `image/jpeg`, `image/png`, `image/webp`, `image/gif`.
- The server validates magic bytes (file signature), rejecting disguised non-image files even if given an image extension.

### 4.2 Example Upload Code (Fetch)

```typescript
export async function uploadMediaAsset(file: File, accessToken: string) {
  const formData = new FormData();
  formData.append("file", file);

  const response = await fetch(`${API_BASE_URL}/api/v1/admin/media`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      // Do NOT explicitly set Content-Type header here;
      // fetch will automatically set multipart/form-data with the correct boundary!
    },
    body: formData,
  });

  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.message || "Upload failed");
  }

  return await response.json();
}
```

### 4.3 Asset URL Resolution

The backend returns file metadata containing a `url` attribute:

- **Local Storage (`STORAGE_DRIVER=local`)**: Returns `/uploads/<filename>`. Prepend your API base URL (e.g., `http://localhost:8080/uploads/avatar_01ARZ...png`).
- **Cloud Storage (`STORAGE_DRIVER=s3` / `r2`)**: Returns the full public CDN URL (e.g., `https://media.yourdomain.com/avatar_01ARZ...png`).

---

## 5. Rich Content & Embedded Media Synchronization

When creating or updating articles (`POST /api/v1/admin/articles` and `PUT /api/v1/admin/articles/{id}`):

1. **HTML Content Sanitization**: The server processes `content` through the `bluemonday` UGC policy, stripping unsafe scripts, iframes, and malicious attributes.
2. **Embedded Media Tracking**: When inserting images into the rich text editor (e.g., TipTap, Quill, Lexical), include the media asset's ULID in the `data-media-id` attribute:
   ```html
   <img
     src="https://media.yourdomain.com/media_01ARZ...webp"
     data-media-id="01ARZ3NDEKTSV4RRFFQ69G5FAV"
     alt="System Architecture"
   />
   ```
   The backend DOM parser automatically extracts all `data-media-id` values and synchronizes the relational `article_media` pivot table. This ensures the media asset cannot be deleted while in use.

---

## 6. Public Visitor Analytics Ingestion

To collect privacy-preserving engagement metrics without cookies or consent banners:

- **Endpoint:** `POST /api/v1/articles/{id}/analytics`
- **Payload:**
  ```json
  {
    "event_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "event_type": "read",
    "reading_time_seconds": 120,
    "scroll_depth": 85
  }
  ```
- **Supported `event_type` Values:** `view`, `read`, `scroll`, `share`.
- **Idempotency:** Always generate a client-side ULID or UUID for `event_id`. If network jitter causes duplicate requests, the server uses `event_id` to prevent duplicate view increments.
- **Privacy:** Visitor identification is computed on the server via `SHA-256(IP + UserAgent + MonthlySalt)`. No personally identifiable information (PII) is stored.
