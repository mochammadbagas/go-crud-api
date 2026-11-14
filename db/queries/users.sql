-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, role, status, email_verified_at, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, role, status, email_verified_at, created_at, updated_at
FROM users
WHERE username = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, role, status, email_verified_at, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, role, status, email_verified_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateUsername :exec
UPDATE users
SET username = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateStatus :exec
UPDATE users
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: VerifyUserEmail :exec
UPDATE users
SET email_verified_at = ?, status = 'active', updated_at = ?
WHERE id = ?;

-- name: UpdateUserEmail :exec
UPDATE users
SET email = ?, email_verified_at = ?, updated_at = ?
WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
