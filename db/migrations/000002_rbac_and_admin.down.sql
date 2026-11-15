-- Migration: 000002_rbac_and_admin.down.sql

SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
ALTER TABLE users MODIFY COLUMN role ENUM('admin', 'user') NOT NULL DEFAULT 'user';
ALTER TABLE users MODIFY COLUMN status ENUM('active', 'pending_verification', 'suspended') NOT NULL DEFAULT 'active';
SET FOREIGN_KEY_CHECKS = 1;
