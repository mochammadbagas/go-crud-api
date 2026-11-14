-- Migration: 000001_init_schema.up.sql
-- Description: Initialize schema with ULID identifiers for Personal Go Starter Kit

CREATE TABLE IF NOT EXISTS users (
    id CHAR(26) NOT NULL,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'user') NOT NULL DEFAULT 'user',
    status ENUM('active', 'pending_verification', 'suspended') NOT NULL DEFAULT 'active',
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

CREATE TABLE IF NOT EXISTS categories (
    id CHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS posts (
    id CHAR(26) NOT NULL,
    type ENUM('blog', 'article', 'journal') NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    excerpt TEXT NULL,
    content LONGTEXT NOT NULL,
    cover_image VARCHAR(500) NULL,
    category_id CHAR(26) NULL,
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
    CONSTRAINT fk_posts_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT chk_posts_status_published CHECK (status <> 'published' OR published_at IS NOT NULL),
    INDEX idx_posts_type_status_published (type, status, published_at),
    INDEX idx_posts_status_published (status, published_at),
    INDEX idx_posts_category_id (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS post_views (
    id CHAR(26) NOT NULL,
    post_id CHAR(26) NOT NULL,
    views BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_post_views_post_id (post_id),
    CONSTRAINT fk_post_views_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
