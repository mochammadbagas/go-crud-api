-- name: GetArticleByID :one
SELECT 
    a.id, a.author_id, a.category_id, a.featured_image_id, a.og_image_id,
    a.title, a.slug, a.excerpt, a.content, a.status, a.is_featured,
    a.scheduled_at, a.published_at, a.meta_title, a.meta_description,
    a.canonical_url, a.robots_index, a.robots_follow, a.meta,
    a.created_at, a.updated_at, a.deleted_at,
    u.username AS author_username,
    c.name AS category_name, c.slug AS category_slug,
    mf.url AS featured_image_url,
    mo.url AS og_image_url,
    COALESCE((SELECT SUM(total_views) FROM article_daily_analytics WHERE article_id = a.id), 0) AS total_views
FROM articles a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN categories c ON a.category_id = c.id
LEFT JOIN media mf ON a.featured_image_id = mf.id
LEFT JOIN media mo ON a.og_image_id = mo.id
WHERE a.id = ? AND a.deleted_at IS NULL
LIMIT 1;

-- name: GetPublishedArticleBySlug :one
SELECT 
    a.id, a.author_id, a.category_id, a.featured_image_id, a.og_image_id,
    a.title, a.slug, a.excerpt, a.content, a.status, a.is_featured,
    a.scheduled_at, a.published_at, a.meta_title, a.meta_description,
    a.canonical_url, a.robots_index, a.robots_follow, a.meta,
    a.created_at, a.updated_at, a.deleted_at,
    u.username AS author_username,
    c.name AS category_name, c.slug AS category_slug,
    mf.url AS featured_image_url,
    mo.url AS og_image_url,
    COALESCE((SELECT SUM(total_views) FROM article_daily_analytics WHERE article_id = a.id), 0) AS total_views
FROM articles a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN categories c ON a.category_id = c.id
LEFT JOIN media mf ON a.featured_image_id = mf.id
LEFT JOIN media mo ON a.og_image_id = mo.id
WHERE a.slug = ? AND a.status = 'published' AND a.deleted_at IS NULL
LIMIT 1;

-- name: GetArticleBySlug :one
SELECT 
    a.id, a.author_id, a.category_id, a.featured_image_id, a.og_image_id,
    a.title, a.slug, a.excerpt, a.content, a.status, a.is_featured,
    a.scheduled_at, a.published_at, a.meta_title, a.meta_description,
    a.canonical_url, a.robots_index, a.robots_follow, a.meta,
    a.created_at, a.updated_at, a.deleted_at,
    u.username AS author_username,
    c.name AS category_name, c.slug AS category_slug,
    mf.url AS featured_image_url,
    mo.url AS og_image_url,
    COALESCE((SELECT SUM(total_views) FROM article_daily_analytics WHERE article_id = a.id), 0) AS total_views
FROM articles a
LEFT JOIN users u ON a.author_id = u.id
LEFT JOIN categories c ON a.category_id = c.id
LEFT JOIN media mf ON a.featured_image_id = mf.id
LEFT JOIN media mo ON a.og_image_id = mo.id
WHERE a.slug = ? AND a.deleted_at IS NULL
LIMIT 1;

-- name: CreateArticle :exec
INSERT INTO articles (
    id, author_id, category_id, featured_image_id, og_image_id,
    title, slug, excerpt, content, status, is_featured,
    scheduled_at, published_at, meta_title, meta_description,
    canonical_url, robots_index, robots_follow, meta,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateArticle :exec
UPDATE articles
SET 
    category_id = ?,
    featured_image_id = ?,
    og_image_id = ?,
    title = ?,
    slug = ?,
    excerpt = ?,
    content = ?,
    status = ?,
    is_featured = ?,
    scheduled_at = ?,
    published_at = ?,
    meta_title = ?,
    meta_description = ?,
    canonical_url = ?,
    robots_index = ?,
    robots_follow = ?,
    meta = ?,
    updated_at = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: SoftDeleteArticle :exec
UPDATE articles
SET deleted_at = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: HardDeleteArticle :exec
DELETE FROM articles
WHERE id = ?;

-- name: PublishScheduledArticles :exec
UPDATE articles
SET status = 'published', published_at = NOW(), updated_at = NOW()
WHERE status = 'scheduled' AND scheduled_at <= NOW() AND deleted_at IS NULL;

-- name: CountArticles :one
SELECT COUNT(*) FROM articles
WHERE deleted_at IS NULL;
