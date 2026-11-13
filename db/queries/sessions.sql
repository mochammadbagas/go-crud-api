-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, expires_at, created_at)
VALUES (?, ?, ?, ?);

-- name: GetSessionByID :one
SELECT s.id, s.user_id, s.expires_at, s.created_at, u.name, u.email
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.id = ? AND s.expires_at > ?
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= ?;

