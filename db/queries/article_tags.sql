-- name: GetTagsByArticleID :many
SELECT t.id, t.name, t.slug, t.created_at, t.updated_at
FROM tags t
INNER JOIN article_tags at ON t.id = at.tag_id
WHERE at.article_id = ?
ORDER BY t.name ASC;

-- name: AddArticleTag :exec
INSERT IGNORE INTO article_tags (article_id, tag_id, created_at)
VALUES (?, ?, ?);

-- name: DeleteArticleTags :exec
DELETE FROM article_tags
WHERE article_id = ?;
