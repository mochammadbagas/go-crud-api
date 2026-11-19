-- Migration: 000001_init_schema.up.sql
-- Description: Unified Schema with ULID identifiers, RBAC, Article Management, Media, Analytics & Settings

-- 1. Users Table
CREATE TABLE IF NOT EXISTS users (
    id CHAR(26) NOT NULL,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('super_admin', 'admin', 'user') NOT NULL DEFAULT 'user',
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

-- 2. User Profiles Table
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

-- 3. Refresh Tokens Table (Stateful Sessions with RTR & Token Family)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    token_family VARCHAR(64) NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    user_agent VARCHAR(255) NULL,
    ip_address VARCHAR(45) NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_refresh_tokens_lookup (token_hash),
    INDEX idx_refresh_tokens_family (token_family),
    INDEX idx_refresh_tokens_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 4. Auth Tokens Table (One-Time Verification, Password Reset, Email Change)
CREATE TABLE IF NOT EXISTS auth_tokens (
    id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    token_type ENUM('password_reset', 'email_verification', 'email_change') NOT NULL,
    payload VARCHAR(255) NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_auth_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_auth_tokens_lookup (token_hash, token_type),
    INDEX idx_auth_tokens_user (user_id),
    INDEX idx_auth_tokens_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 5. RBAC: Roles Table
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

-- 6. RBAC: Permissions Table
CREATE TABLE IF NOT EXISTS permissions (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_permissions_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 7. RBAC: Role Permissions Pivot Table
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id CHAR(26) NOT NULL,
    permission_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    INDEX idx_rp_permission_id (permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 8. RBAC: User Roles Pivot Table
CREATE TABLE IF NOT EXISTS user_roles (
    user_id CHAR(26) NOT NULL,
    role_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    INDEX idx_ur_role_id (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 9. Categories Table
CREATE TABLE IF NOT EXISTS categories (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    description TEXT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 10. Tags Table
CREATE TABLE IF NOT EXISTS tags (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_tags_slug (slug),
    INDEX idx_tags_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 11. Media Assets Catalog Table
CREATE TABLE IF NOT EXISTS media (
    id CHAR(26) NOT NULL,
    uploaded_by CHAR(26) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    storage_disk VARCHAR(255) NOT NULL,
    storage_key VARCHAR(1024) NOT NULL,
    url VARCHAR(500) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    width INT NULL,
    height INT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_media_uploaded_by (uploaded_by),
    INDEX idx_media_mime_type (mime_type),
    CONSTRAINT fk_media_user FOREIGN KEY (uploaded_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 12. Articles Table
CREATE TABLE IF NOT EXISTS articles (
    id CHAR(26) NOT NULL,
    author_id CHAR(26) NOT NULL,
    category_id CHAR(26) NULL,
    featured_image_id CHAR(26) NULL,
    og_image_id CHAR(26) NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    excerpt TEXT NULL,
    content LONGTEXT NOT NULL,
    status ENUM('draft', 'scheduled', 'published', 'archived') NOT NULL DEFAULT 'draft',
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    scheduled_at DATETIME NULL,
    published_at DATETIME NULL,
    meta_title VARCHAR(255) NULL,
    meta_description TEXT NULL,
    canonical_url VARCHAR(500) NULL,
    robots_index BOOLEAN NOT NULL DEFAULT TRUE,
    robots_follow BOOLEAN NOT NULL DEFAULT TRUE,
    meta JSON NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_articles_slug (slug),
    CONSTRAINT fk_articles_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_articles_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_articles_featured_image FOREIGN KEY (featured_image_id) REFERENCES media(id) ON DELETE SET NULL,
    CONSTRAINT fk_articles_og_image FOREIGN KEY (og_image_id) REFERENCES media(id) ON DELETE SET NULL,
    INDEX idx_articles_status_scheduled (status, scheduled_at),
    INDEX idx_articles_author (author_id),
    INDEX idx_articles_category (category_id),
    INDEX idx_articles_featured (is_featured),
    INDEX idx_articles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 13. Article Tags Junction Table
CREATE TABLE IF NOT EXISTS article_tags (
    article_id CHAR(26) NOT NULL,
    tag_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (article_id, tag_id),
    INDEX idx_article_tags_tag (tag_id),
    CONSTRAINT fk_article_tags_article FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    CONSTRAINT fk_article_tags_tag FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 14. Article Media Junction Table (Tracking embedded images)
CREATE TABLE IF NOT EXISTS article_media (
    article_id CHAR(26) NOT NULL,
    media_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (article_id, media_id),
    INDEX idx_article_media_media (media_id),
    CONSTRAINT fk_article_media_article FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    CONSTRAINT fk_article_media_media FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 15. Article Revisions Table (Version history snapshots)
CREATE TABLE IF NOT EXISTS article_revisions (
    id CHAR(26) NOT NULL,
    article_id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    snapshot JSON NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_article_revisions_article (article_id),
    INDEX idx_article_revisions_created (created_at),
    CONSTRAINT fk_article_revisions_article FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    CONSTRAINT fk_article_revisions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 16. Article Related Articles Table
CREATE TABLE IF NOT EXISTS article_related (
    article_id CHAR(26) NOT NULL,
    related_article_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (article_id, related_article_id),
    INDEX idx_article_related_target (related_article_id),
    CONSTRAINT fk_article_related_source FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    CONSTRAINT fk_article_related_target FOREIGN KEY (related_article_id) REFERENCES articles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 17. Article Analytics Events Table (Raw Visitor Engagement Stream)
CREATE TABLE IF NOT EXISTS article_analytics_events (
    id BIGINT NOT NULL AUTO_INCREMENT,
    article_id CHAR(26) NOT NULL,
    event_id VARCHAR(100) NOT NULL,
    visitor_id VARCHAR(64) NOT NULL,
    session_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    reading_time_seconds INT NOT NULL DEFAULT 0,
    scroll_depth INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_analytics_event_id (event_id),
    INDEX idx_analytics_article_date (article_id, created_at),
    INDEX idx_analytics_visitor (visitor_id),
    CONSTRAINT fk_analytics_events_article FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 18. Article Daily Analytics Table (Aggregated Metrics)
CREATE TABLE IF NOT EXISTS article_daily_analytics (
    id BIGINT NOT NULL AUTO_INCREMENT,
    article_id CHAR(26) NOT NULL,
    analytics_date DATE NOT NULL,
    total_views BIGINT NOT NULL DEFAULT 0,
    unique_visitors BIGINT NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 0,
    total_reading_time BIGINT NOT NULL DEFAULT 0,
    average_scroll_depth DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_article_daily_date (article_id, analytics_date),
    INDEX idx_daily_analytics_date (analytics_date),
    CONSTRAINT fk_daily_analytics_article FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 19. Global Application Settings Table
CREATE TABLE IF NOT EXISTS settings (
    id CHAR(26) NOT NULL,
    setting_key VARCHAR(100) NOT NULL,
    setting_value JSON NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_settings_key (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
