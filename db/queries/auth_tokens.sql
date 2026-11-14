-- name: CreateAuthToken :exec
INSERT INTO auth_tokens (id, user_id, token_hash, token_type, payload, is_used, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAuthTokenByHashAndType :one
SELECT id, user_id, token_hash, token_type, payload, is_used, expires_at, created_at
FROM auth_tokens
WHERE token_hash = ? AND token_type = ?
LIMIT 1;

-- name: MarkAuthTokenAsUsed :exec
UPDATE auth_tokens
SET is_used = TRUE
WHERE id = ?;

-- name: InvalidateUserTokensByType :exec
UPDATE auth_tokens
SET is_used = TRUE
WHERE user_id = ? AND token_type = ?;

-- name: DeleteExpiredOrUsedAuthTokens :exec
DELETE FROM auth_tokens
WHERE expires_at < ? OR is_used = TRUE;
