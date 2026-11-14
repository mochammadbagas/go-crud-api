-- name: GetPostViews :one
SELECT id, post_id, views, updated_at
FROM post_views
WHERE post_id = ?
LIMIT 1;

-- name: IncrementPostViews :exec
INSERT INTO post_views (id, post_id, views, updated_at)
VALUES (?, ?, 1, ?)
ON DUPLICATE KEY UPDATE
    views = views + 1,
    updated_at = VALUES(updated_at);
