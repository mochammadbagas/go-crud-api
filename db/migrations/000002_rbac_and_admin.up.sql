-- Migration: 000002_rbac_and_admin.up.sql
-- Description: Add RBAC (roles, permissions, role_permissions, user_roles) and update user status

ALTER TABLE users MODIFY COLUMN role ENUM('super_admin', 'admin', 'user') NOT NULL DEFAULT 'user';
ALTER TABLE users MODIFY COLUMN status ENUM('active', 'inactive', 'pending_verification', 'suspended') NOT NULL DEFAULT 'active';

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
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    INDEX idx_rp_permission_id (permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_roles (
    user_id CHAR(26) NOT NULL,
    role_id CHAR(26) NOT NULL,
    created_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    INDEX idx_ur_role_id (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
