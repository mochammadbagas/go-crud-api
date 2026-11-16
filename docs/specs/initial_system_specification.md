# Historical System Specification & Build-Phase Reference

**Document Type:** Historical Build-Phase Specification (Archived)  
**Status:** Archived / Superseded by Active Documentation  
**Active Documentation References:**
- For current architecture and layering, see [docs/architecture.md](../architecture.md).
- For step-by-step module extension, see [docs/how_to_add_new_module.md](../how_to_add_new_module.md).
- For security, JWT RTR, and RBAC, see [docs/security_and_rbac.md](../security_and_rbac.md).
- For production deployment, see [docs/deployment.md](../deployment.md).
- For Go idioms and educational walkthrough, see [docs/learning_guide.md](../learning_guide.md).

---

## 1. Overview & Project Vision

This document consolidates the original technical requirements, database design, and architecture decisions made during the initial build phase of the **Go REST API Starter Kit (`go-crud-api`)**.

The initial project goal was to build a production-grade, enterprise-ready Go RESTful backend boilerplate providing:
- **Decentralized Entity Identity (ULID)**: Standardizing on 26-character Crockford Base32 identifiers across all tables (`CHAR(26)`), avoiding auto-increment sequence locks, sequential scraping, and 53-bit JavaScript number truncation.
- **Modern Authentication**: JWT Access Tokens (HS256) paired with stateful Refresh Token Rotation (RTR) and Token Family tracking for automatic session reuse detection.
- **Granular RBAC**: Role-Based Access Control supporting multi-role assignment per user, `resource.action` syntax, and wildcard bypass for super administrators.
- **Administrative Safeguards**: Permanent domain protection for the last remaining active super administrator account.
- **Pluggable Architecture**: Modular drivers for file storage (Local disk vs Cloudflare R2 / AWS S3) and transactional email (Console Log vs Resend API).
- **Multi-Input Field Validation**: Aggregated error collection across all payload fields without early exit.

---

## 2. Complete Relational Database Schema (MySQL 8.0+)

The relational layer is managed via embedded `golang-migrate` DDL scripts:

```sql
-- 1. Core Identity & User Profiles
CREATE TABLE IF NOT EXISTS users (
    id CHAR(26) NOT NULL,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
    status ENUM('active', 'inactive', 'pending_verification', 'suspended') NOT NULL DEFAULT 'active',
    email_verified_at DATETIME NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_username (username),
    UNIQUE KEY uq_users_email (email),
    INDEX idx_users_status (status),
    INDEX idx_users_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id CHAR(26) NOT NULL,
    full_name VARCHAR(100) NOT NULL DEFAULT '',
    bio TEXT NULL,
    phone_number VARCHAR(20) NULL,
    avatar_url VARCHAR(500) NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (user_id),
    CONSTRAINT fk_user_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 2. Stateful Sessions & Ephemeral Auth Tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    token_family VARCHAR(64) NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    user_agent VARCHAR(500) NULL,
    ip_address VARCHAR(45) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_refresh_tokens_hash (token_hash),
    INDEX idx_refresh_tokens_family (token_family),
    INDEX idx_refresh_tokens_user (user_id),
    INDEX idx_refresh_tokens_expires (expires_at),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auth_tokens (
    id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    token_type ENUM('email_verification', 'password_reset', 'email_change') NOT NULL,
    payload VARCHAR(255) NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_auth_tokens_hash_type (token_hash, token_type),
    INDEX idx_auth_tokens_user_type (user_id, token_type),
    INDEX idx_auth_tokens_expires (expires_at),
    CONSTRAINT fk_auth_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 3. Granular Role-Based Access Control (RBAC)
CREATE TABLE IF NOT EXISTS roles (
    id CHAR(26) NOT NULL,
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS permissions (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_permissions_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id CHAR(26) NOT NULL,
    permission_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_rp_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_roles (
    user_id CHAR(26) NOT NULL,
    role_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_ur_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_ur_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4. Reference Content Management Module
CREATE TABLE IF NOT EXISTS categories (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS posts (
    id CHAR(26) NOT NULL,
    type ENUM('blog', 'article', 'journal') NOT NULL DEFAULT 'article',
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    excerpt TEXT NULL,
    content MEDIUMTEXT NOT NULL,
    cover_image VARCHAR(500) NULL,
    category_id CHAR(26) NOT NULL,
    tags JSON NULL,
    meta_title VARCHAR(255) NULL,
    meta_description TEXT NULL,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    status ENUM('draft', 'published') NOT NULL DEFAULT 'draft',
    published_at DATETIME NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_posts_slug (slug),
    INDEX idx_posts_type (type),
    INDEX idx_posts_status_published (status, published_at),
    INDEX idx_posts_category (category_id),
    INDEX idx_posts_featured (is_featured),
    CONSTRAINT fk_posts_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS post_views (
    id CHAR(26) NOT NULL,
    post_id CHAR(26) NOT NULL,
    views BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_post_views_post (post_id),
    CONSTRAINT fk_post_views_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

---

## 3. Technology Stack Decisions

| Decision | Selected Technology | Primary Architectural Justification |
| :--- | :--- | :--- |
| **Language** | Go (Golang) 1.25+ | Compiled execution speed, low memory footprint, first-class standard library concurrency. |
| **Routing** | Chi Router v5 | Idiomatic compatibility with standard `net/http`, lightweight sub-routing, zero global state. |
| **SQL Handling** | SQLC | Compile-time type-safety from pure SQL without the runtime reflection or N+1 queries of ORMs. |
| **Identity** | ULID | 128-bit sortable identity without sequential integer enumeration or UUID index fragmentation. |
| **Auth Cryptography** | Argon2id + HS256 JWT | RFC 9106 memory-hard password protection combined with stateless bearer token verification. |
| **Database Migrations** | golang-migrate | Embedded migration scripts (`//go:embed`) executing automatically on application startup. |
| **Media Storage** | Pluggable Adapter (Local / S3 / R2) | Clean isolation behind a single `Storage` interface with automated orphan file cleanup. |
| **Transactional Email** | Pluggable Adapter (Log / Resend) | Zero-configuration console development mode with seamless production API switching. |
