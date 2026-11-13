-- name: ListCategories :many
SELECT id, name, slug, created_at, updated_at
FROM categories
ORDER BY name ASC;

-- name: GetCategoryByID :one
SELECT id, name, slug, created_at, updated_at
FROM categories
WHERE id = ?
LIMIT 1;

-- name: GetCategoryBySlug :one
SELECT id, name, slug, created_at, updated_at
FROM categories
WHERE slug = ?
LIMIT 1;

-- name: CreateCategory :execresult
INSERT INTO categories (name, slug, created_at, updated_at)
VALUES (?, ?, ?, ?);

-- name: UpdateCategory :exec
UPDATE categories
SET name = ?, slug = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

