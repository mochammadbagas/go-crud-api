-- name: ListCategories :many
SELECT 
    c.id, c.name, c.slug, c.description, c.created_at, c.updated_at,
    COUNT(a.id) AS article_count
FROM categories c
LEFT JOIN articles a ON c.id = a.category_id AND a.deleted_at IS NULL
GROUP BY c.id
ORDER BY c.name ASC;

-- name: GetCategoryByID :one
SELECT id, name, slug, description, created_at, updated_at
FROM categories
WHERE id = ?
LIMIT 1;

-- name: GetCategoryBySlug :one
SELECT id, name, slug, description, created_at, updated_at
FROM categories
WHERE slug = ?
LIMIT 1;

-- name: CreateCategory :exec
INSERT INTO categories (id, name, slug, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateCategory :exec
UPDATE categories
SET name = ?, slug = ?, description = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

-- name: CountArticlesByCategoryID :one
SELECT COUNT(*) FROM articles
WHERE category_id = ? AND deleted_at IS NULL;

-- name: DetachCategoryFromSoftDeletedArticles :exec
UPDATE articles
SET category_id = NULL
WHERE category_id = ? AND deleted_at IS NOT NULL;
