-- name: CreatePermission :exec
INSERT INTO permissions (id, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetPermissionByID :one
SELECT id, name, description, created_at, updated_at
FROM permissions
WHERE id = ?
LIMIT 1;

-- name: GetPermissionByName :one
SELECT id, name, description, created_at, updated_at
FROM permissions
WHERE name = ?
LIMIT 1;

-- name: ListPermissions :many
SELECT id, name, description, created_at, updated_at
FROM permissions
ORDER BY name ASC;

-- name: UpdatePermission :exec
UPDATE permissions
SET description = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE id = ?;

-- name: CountPermissions :one
SELECT COUNT(*) FROM permissions;
