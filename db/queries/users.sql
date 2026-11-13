-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: CreateUser :execresult
INSERT INTO users (name, email, password_hash, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

