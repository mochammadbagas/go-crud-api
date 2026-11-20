-- name: ListTags :many
SELECT 
    t.id, t.name, t.slug, t.created_at, t.updated_at,
    COUNT(a.id) AS usage_count
FROM tags t
LEFT JOIN article_tags at ON t.id = at.tag_id
LEFT JOIN articles a ON at.article_id = a.id AND a.deleted_at IS NULL
GROUP BY t.id
ORDER BY t.name ASC;

-- name: GetTagByID :one
SELECT id, name, slug, created_at, updated_at
FROM tags
WHERE id = ?
LIMIT 1;

-- name: GetTagBySlug :one
SELECT id, name, slug, created_at, updated_at
FROM tags
WHERE slug = ?
LIMIT 1;

-- name: GetTagByName :one
SELECT id, name, slug, created_at, updated_at
FROM tags
WHERE name = ?
LIMIT 1;

-- name: CreateTag :exec
INSERT INTO tags (id, name, slug, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateTag :exec
UPDATE tags
SET name = ?, slug = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = ?;

-- name: CountTags :one
SELECT COUNT(*) FROM tags;
