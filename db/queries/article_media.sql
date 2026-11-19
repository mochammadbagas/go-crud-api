-- name: GetMediaByArticleID :many
SELECT 
    m.id, m.uploaded_by, m.filename, m.storage_disk, m.storage_key,
    m.url, m.mime_type, m.size, m.width, m.height, m.created_at, m.updated_at
FROM media m
INNER JOIN article_media am ON m.id = am.media_id
WHERE am.article_id = ?
ORDER BY am.created_at ASC;

-- name: AddArticleMedia :exec
INSERT IGNORE INTO article_media (article_id, media_id, created_at)
VALUES (?, ?, ?);

-- name: DeleteArticleMedia :exec
DELETE FROM article_media
WHERE article_id = ?;
