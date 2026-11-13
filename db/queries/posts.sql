-- name: GetPostByID :one
SELECT 
    p.id, p.type, p.title, p.slug, p.excerpt, p.content, p.cover_image,
    p.category_id, p.tags, p.meta_title, p.meta_description,
    p.is_featured, p.status, p.published_at, p.created_at, p.updated_at,
    c.name AS category_name, c.slug AS category_slug,
    COALESCE(pv.views, 0) AS views
FROM posts p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN post_views pv ON p.id = pv.post_id
WHERE p.id = ?
LIMIT 1;

-- name: GetPublishedPostBySlug :one
SELECT 
    p.id, p.type, p.title, p.slug, p.excerpt, p.content, p.cover_image,
    p.category_id, p.tags, p.meta_title, p.meta_description,
    p.is_featured, p.status, p.published_at, p.created_at, p.updated_at,
    c.name AS category_name, c.slug AS category_slug,
    COALESCE(pv.views, 0) AS views
FROM posts p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN post_views pv ON p.id = pv.post_id
WHERE p.slug = ? AND p.status = 'published'
LIMIT 1;

-- name: GetPostBySlug :one
SELECT 
    p.id, p.type, p.title, p.slug, p.excerpt, p.content, p.cover_image,
    p.category_id, p.tags, p.meta_title, p.meta_description,
    p.is_featured, p.status, p.published_at, p.created_at, p.updated_at,
    c.name AS category_name, c.slug AS category_slug,
    COALESCE(pv.views, 0) AS views
FROM posts p
LEFT JOIN categories c ON p.category_id = c.id
LEFT JOIN post_views pv ON p.id = pv.post_id
WHERE p.slug = ?
LIMIT 1;

-- name: CreatePost :execresult
INSERT INTO posts (
    type, title, slug, excerpt, content, cover_image,
    category_id, tags, meta_title, meta_description,
    is_featured, status, published_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePost :exec
UPDATE posts
SET 
    type = ?,
    title = ?,
    slug = ?,
    excerpt = ?,
    content = ?,
    cover_image = ?,
    category_id = ?,
    tags = ?,
    meta_title = ?,
    meta_description = ?,
    is_featured = ?,
    status = ?,
    published_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeletePost :exec
DELETE FROM posts
WHERE id = ?;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts;

