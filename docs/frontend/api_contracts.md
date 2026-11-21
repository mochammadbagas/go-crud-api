# Complete REST API Contract Directory

**Document Type:** Formal Endpoint Contract & Authorization Directory  
**Target Audience:** Frontend Developers, Mobile Engineers, API Consumers, and QA Engineers  
**Base URL:** `http://localhost:8080` (Development) / `https://api.yourdomain.com` (Production)

---

## 0. System Health & Probes

| Method | Endpoint        | Auth   | Purpose & Responses                                                                        |
| :----- | :-------------- | :----- | :----------------------------------------------------------------------------------------- |
| `GET`  | `/health`       | Public | Deep health probe verifying MySQL connection pool (`200 OK` or `503 Service Unavailable`). |
| `GET`  | `/health/live`  | Public | Instant container liveness probe for Docker / Kubernetes (`200 OK`).                       |
| `GET`  | `/health/ready` | Public | Readiness probe for ingress traffic routing (`200 OK` / `503 Service Unavailable`).        |

---

## 1. Authentication (`/api/v1/auth`)

Sensitive public endpoints are protected by IP rate limiting (10 requests/minute).

| Method | Endpoint                           | Headers / Auth                  | Request Body                                       | Response (Success)                                                        |
| :----- | :--------------------------------- | :------------------------------ | :------------------------------------------------- | :------------------------------------------------------------------------ |
| `POST` | `/api/v1/auth/register`            | None                            | `{ "username", "email", "password", "full_name" }` | `201 Created` + `{ "message", "data": { user } }`                         |
| `POST` | `/api/v1/auth/login`               | None                            | `{ "email", "password" }`                          | `200 OK` + Access Token (JSON) & Refresh Token (`HttpOnly` Cookie + JSON) |
| `POST` | `/api/v1/auth/refresh`             | Cookie or `X-Refresh-Token`     | `{ "refresh_token"?: "..." }`                      | `200 OK` + New Access Token & Rotated Refresh Token                       |
| `POST` | `/api/v1/auth/logout`              | `Authorization: Bearer <token>` | None                                               | `200 OK` + `{ "message": "logged out successfully" }` (Cookie cleared)    |
| `POST` | `/api/v1/auth/change-password`     | `Authorization: Bearer <token>` | `{ "current_password", "new_password" }`           | `200 OK` (Revokes all other active sessions)                              |
| `POST` | `/api/v1/auth/forgot-password`     | None                            | `{ "email" }`                                      | `200 OK` (Anti-enumeration; sends reset email)                            |
| `POST` | `/api/v1/auth/reset-password`      | None                            | `{ "token", "new_password" }`                      | `200 OK` (Revokes all active sessions)                                    |
| `POST` | `/api/v1/auth/verify-email`        | None                            | `{ "token" }` or query `?token=...`                | `200 OK` (Activates account)                                              |
| `POST` | `/api/v1/auth/resend-verification` | None                            | `{ "email" }`                                      | `200 OK` (Anti-enumeration)                                               |

---

## 2. User Profile Management (`/api/v1/users`)

All endpoints require valid `Authorization: Bearer <token>` unless marked public.

| Method   | Endpoint                             | Auth         | Request Body / Query                       | Description                                                |
| :------- | :----------------------------------- | :----------- | :----------------------------------------- | :--------------------------------------------------------- |
| `GET`    | `/api/v1/users/me`                   | Bearer Token | None                                       | Returns authenticated user identity, roles, and profile.   |
| `PUT`    | `/api/v1/users/profile`              | Bearer Token | `{ "full_name", "bio"?, "phone_number"? }` | Updates profile fields.                                    |
| `PUT`    | `/api/v1/users/username`             | Bearer Token | `{ "username" }`                           | Updates username with uniqueness validation.               |
| `GET`    | `/api/v1/users/status`               | Bearer Token | None                                       | Returns verification and active status.                    |
| `POST`   | `/api/v1/users/avatar`               | Bearer Token | `multipart/form-data` (`file`)             | Uploads avatar picture (max 5MB, JPG/PNG/WEBP/GIF).        |
| `DELETE` | `/api/v1/users/avatar`               | Bearer Token | None                                       | Deletes existing avatar and cleans up file storage.        |
| `POST`   | `/api/v1/users/email/request-change` | Bearer Token | `{ "new_email", "password" }`              | Sends confirmation token to the new email address.         |
| `POST`   | `/api/v1/users/email/confirm-change` | Public       | `{ "token" }` or query `?token=...`        | Confirms new email address; invalidates previous sessions. |

---

## 3. Public Articles & Taxonomy

Public reader endpoints optimized for client applications and SEO.

| Method | Endpoint                             | Query Parameters                                          | Description                                                                                  |
| :----- | :----------------------------------- | :-------------------------------------------------------- | :------------------------------------------------------------------------------------------- |
| `GET`  | `/api/v1/articles`                   | `?page=1&per_page=20&category=&tag=&search=&is_featured=` | Lists published articles with pagination.                                                    |
| `GET`  | `/api/v1/articles/{slug}`            | None                                                      | Retrieves single published article by slug with author, category, tags, and media relations. |
| `POST` | `/api/v1/articles/{id}/analytics`    | Rate Limited (60/min)                                     | `{ "event_id", "event_type", "reading_time_seconds", "scroll_depth" }`                       |
| `GET`  | `/api/v1/categories`                 | None                                                      | Lists all categories with total article counts.                                              |
| `GET`  | `/api/v1/categories/{slug}/articles` | `?page=1&per_page=20`                                     | Lists published articles under category slug.                                                |
| `GET`  | `/api/v1/tags`                       | None                                                      | Lists all tags with article counts.                                                          |
| `GET`  | `/api/v1/tags/{slug}`                | None                                                      | Retrieves tag details by slug.                                                               |
| `GET`  | `/articles/{slug}`                   | None                                                      | SEO alias directly routing to article details.                                               |
| `GET`  | `/blog/{slug}`, `/journal/{slug}`    | None                                                      | Vanity aliases for articles.                                                                 |

---

## 4. Article & Content Administration (`/api/v1/admin`)

Requires valid JWT Access Token and specific granular permissions.

| Method   | Endpoint                                | Required Permission | Description                                                                     |
| :------- | :-------------------------------------- | :------------------ | :------------------------------------------------------------------------------ |
| `GET`    | `/api/v1/admin/articles`                | `articles.read`     | Lists all articles (draft, scheduled, published, archived).                     |
| `POST`   | `/api/v1/admin/articles`                | `articles.create`   | Creates article (auto XSS HTML sanitize, embedded media extraction).            |
| `GET`    | `/api/v1/admin/articles/{id}`           | `articles.read`     | Retrieves article details by ULID.                                              |
| `PUT`    | `/api/v1/admin/articles/{id}`           | `articles.update`   | Updates article and takes historical snapshot in `article_revisions`.           |
| `DELETE` | `/api/v1/admin/articles/{id}`           | `articles.delete`   | Soft deletes article (`deleted_at = NOW()`).                                    |
| `GET`    | `/api/v1/admin/articles/{id}/revisions` | `articles.read`     | Lists all historical revision snapshots.                                        |
| `GET`    | `/api/v1/admin/categories`              | `categories.manage` | Lists all categories with management metadata.                                  |
| `POST`   | `/api/v1/admin/categories`              | `categories.manage` | Creates a new category.                                                         |
| `PUT`    | `/api/v1/admin/categories/{id}`         | `categories.manage` | Updates category by ULID.                                                       |
| `DELETE` | `/api/v1/admin/categories/{id}`         | `categories.manage` | Deletes category (blocked if category contains articles: `409 Conflict`).       |
| `GET`    | `/api/v1/admin/tags`                    | `tags.read`         | Lists all tags.                                                                 |
| `POST`   | `/api/v1/admin/tags`                    | `tags.manage`       | Creates a new tag.                                                              |
| `PUT`    | `/api/v1/admin/tags/{id}`               | `tags.manage`       | Updates tag by ULID.                                                            |
| `DELETE` | `/api/v1/admin/tags/{id}`               | `tags.manage`       | Deletes tag by ULID.                                                            |
| `POST`   | `/api/v1/admin/media`                   | `media.upload`      | Uploads media asset (`multipart/form-data`, max 10MB, auto dimension sniffing). |
| `GET`    | `/api/v1/admin/media`                   | `media.read`        | Lists media assets with pagination.                                             |
| `GET`    | `/api/v1/admin/media/{id}`              | `media.read`        | Retrieves media metadata by ULID.                                               |
| `DELETE` | `/api/v1/admin/media/{id}`              | `media.delete`      | Deletes media asset (blocked if asset is in use: `409 Conflict`).               |
| `GET`    | `/api/v1/admin/articles/{id}/analytics` | `analytics.read`    | Retrieves daily analytics engagement report by date range.                      |
| `GET`    | `/api/v1/admin/analytics/overview`      | `analytics.read`    | Retrieves global platform engagement totals.                                    |

---

## 5. Role-Based Access Control (`/api/v1/admin/roles` & `/permissions`)

Requires valid JWT Access Token and RBAC management permissions.

| Method   | Endpoint                               | Required Permission  | Description                                      |
| :------- | :------------------------------------- | :------------------- | :----------------------------------------------- |
| `GET`    | `/api/v1/admin/roles`                  | `roles.read`         | Lists all RBAC roles.                            |
| `POST`   | `/api/v1/admin/roles`                  | `roles.create`       | Creates a new role.                              |
| `GET`    | `/api/v1/admin/roles/{id}`             | `roles.read`         | Retrieves role details and assigned permissions. |
| `PUT`    | `/api/v1/admin/roles/{id}`             | `roles.update`       | Updates role name or description.                |
| `PUT`    | `/api/v1/admin/roles/{id}/permissions` | `roles.update`       | Replaces permission bindings array for a role.   |
| `DELETE` | `/api/v1/admin/roles/{id}`             | `roles.delete`       | Deletes role by ULID.                            |
| `GET`    | `/api/v1/admin/permissions`            | `permissions.read`   | Lists all system permissions.                    |
| `POST`   | `/api/v1/admin/permissions`            | `permissions.create` | Registers a new granular permission descriptor.  |
| `GET`    | `/api/v1/admin/permissions/{id}`       | `permissions.read`   | Retrieves permission details.                    |
| `PUT`    | `/api/v1/admin/permissions/{id}`       | `permissions.update` | Updates permission description.                  |
| `DELETE` | `/api/v1/admin/permissions/{id}`       | `permissions.delete` | Deletes permission by ULID.                      |

---

## 6. User Administration (`/api/v1/admin/users`)

Requires valid JWT Access Token and user management permissions. Modifications affecting user active status or role demotions enforce the **Super Admin Safeguard**, blocking actions against the last active `super_admin` with HTTP 403 Forbidden.

| Method   | Endpoint                          | Required Permission | Description                                                                 |
| :------- | :-------------------------------- | :------------------ | :-------------------------------------------------------------------------- |
| `GET`    | `/api/v1/admin/users`             | `users.read`        | Lists users (`?page=&per_page=&search=&status=&role=`).                     |
| `GET`    | `/api/v1/admin/users/{id}`        | `users.read`        | Retrieves user profile, status, and role assignments.                       |
| `PUT`    | `/api/v1/admin/users/{id}`        | `users.update`      | Updates user credentials and profile details.                               |
| `PUT`    | `/api/v1/admin/users/{id}/status` | `users.update`      | Updates status (`active`, `inactive`, `pending_verification`, `suspended`). |
| `PUT`    | `/api/v1/admin/users/{id}/roles`  | `users.update`      | Assigns roles array; auto revokes active sessions to enforce fresh login.   |
| `DELETE` | `/api/v1/admin/users/{id}`        | `users.delete`      | Permanently deletes user and removes avatar file from storage.              |
