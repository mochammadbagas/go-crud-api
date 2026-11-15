-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id, created_at)
VALUES (?, ?, ?);

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = ? AND role_id = ?;

-- name: RemoveAllRolesFromUser :exec
DELETE FROM user_roles
WHERE user_id = ?;

-- name: GetRolesByUserID :many
SELECT r.id, r.name, r.display_name, r.description, r.created_at, r.updated_at
FROM roles r
INNER JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = ?
ORDER BY r.name ASC;

-- name: GetUserRoleNames :many
SELECT r.name
FROM roles r
INNER JOIN user_roles ur ON r.id = ur.role_id
WHERE ur.user_id = ?
ORDER BY r.name ASC;

-- name: GetUserPermissionsByUserID :many
SELECT DISTINCT p.name
FROM permissions p
INNER JOIN role_permissions rp ON p.id = rp.permission_id
INNER JOIN user_roles ur ON rp.role_id = ur.role_id
WHERE ur.user_id = ?
ORDER BY p.name ASC;

-- name: CountActiveSuperAdmins :one
SELECT COUNT(DISTINCT u.id)
FROM users u
INNER JOIN user_roles ur ON u.id = ur.user_id
INNER JOIN roles r ON ur.role_id = r.id
WHERE r.name = 'super_admin' AND u.status = 'active';

-- name: CountActiveSuperAdminsExcludingUser :one
SELECT COUNT(DISTINCT u.id)
FROM users u
INNER JOIN user_roles ur ON u.id = ur.user_id
INNER JOIN roles r ON ur.role_id = r.id
WHERE r.name = 'super_admin' AND u.status = 'active' AND u.id != ?;

-- name: CheckUserHasRole :one
SELECT COUNT(*)
FROM user_roles ur
INNER JOIN roles r ON ur.role_id = r.id
WHERE ur.user_id = ? AND r.name = ?;
