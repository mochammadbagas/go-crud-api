-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, token_family, is_revoked, expires_at, created_at, user_agent, ip_address)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, token_family, is_revoked, expires_at, created_at, user_agent, ip_address
FROM refresh_tokens
WHERE token_hash = ?
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE id = ?;

-- name: RevokeRefreshTokenAtomic :execresult
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE id = ? AND is_revoked = FALSE;

-- name: RevokeTokenFamily :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE token_family = ?;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE user_id = ?;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < ?;
