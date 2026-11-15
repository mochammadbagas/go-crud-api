-- name: CreateRole :exec
INSERT INTO roles (id, name, display_name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetRoleByID :one
SELECT id, name, display_name, description, created_at, updated_at
FROM roles
WHERE id = ?
LIMIT 1;

-- name: GetRoleByName :one
SELECT id, name, display_name, description, created_at, updated_at
FROM roles
WHERE name = ?
LIMIT 1;

-- name: ListRoles :many
SELECT id, name, display_name, description, created_at, updated_at
FROM roles
ORDER BY name ASC;

-- name: UpdateRole :exec
UPDATE roles
SET display_name = ?, description = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = ?;

-- name: CountRoles :one
SELECT COUNT(*) FROM roles;
