-- name: ListMedia :many
SELECT 
    id, uploaded_by, filename, storage_disk, storage_key,
    url, mime_type, size, width, height, created_at, updated_at
FROM media
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: GetMediaByID :one
SELECT 
    id, uploaded_by, filename, storage_disk, storage_key,
    url, mime_type, size, width, height, created_at, updated_at
FROM media
WHERE id = ?
LIMIT 1;

-- name: CreateMedia :exec
INSERT INTO media (
    id, uploaded_by, filename, storage_disk, storage_key,
    url, mime_type, size, width, height, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteMedia :exec
DELETE FROM media
WHERE id = ?;

-- name: CountMedia :one
SELECT COUNT(*) FROM media;

-- name: IsMediaInUse :one
SELECT (
    EXISTS (SELECT 1 FROM articles a WHERE a.featured_image_id = ? AND a.deleted_at IS NULL) OR
    EXISTS (SELECT 1 FROM articles b WHERE b.og_image_id = ? AND b.deleted_at IS NULL) OR
    EXISTS (SELECT 1 FROM article_media m WHERE m.media_id = ?)
) AS in_use;
