-- name: CreateArticleRevision :exec
INSERT INTO article_revisions (id, article_id, user_id, snapshot, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListArticleRevisions :many
SELECT r.id, r.article_id, r.user_id, r.snapshot, r.created_at, u.username, u.email
FROM article_revisions r
INNER JOIN users u ON r.user_id = u.id
WHERE r.article_id = ?
ORDER BY r.created_at DESC;

-- name: GetArticleRevisionByID :one
SELECT r.id, r.article_id, r.user_id, r.snapshot, r.created_at, u.username, u.email
FROM article_revisions r
INNER JOIN users u ON r.user_id = u.id
WHERE r.id = ?
LIMIT 1;
