-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id, created_at)
VALUES (?, ?, ?);

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = ? AND permission_id = ?;

-- name: RemoveAllPermissionsFromRole :exec
DELETE FROM role_permissions
WHERE role_id = ?;

-- name: GetPermissionsByRoleID :many
SELECT p.id, p.name, p.description, p.created_at, p.updated_at
FROM permissions p
INNER JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = ?
ORDER BY p.name ASC;

-- name: GetRolePermissionIDs :many
SELECT permission_id
FROM role_permissions
WHERE role_id = ?;
