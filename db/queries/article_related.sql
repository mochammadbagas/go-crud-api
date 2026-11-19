-- name: GetRelatedArticles :many
SELECT 
    a.id, a.title, a.slug, a.excerpt, a.status, a.published_at, a.created_at,
    m.url AS featured_image_url
FROM articles a
INNER JOIN article_related ar ON a.id = ar.related_article_id
LEFT JOIN media m ON a.featured_image_id = m.id
WHERE ar.article_id = ? AND a.deleted_at IS NULL AND a.status = 'published'
ORDER BY ar.created_at ASC;

-- name: AddRelatedArticle :exec
INSERT IGNORE INTO article_related (article_id, related_article_id, created_at)
VALUES (?, ?, ?);

-- name: DeleteRelatedArticles :exec
DELETE FROM article_related
WHERE article_id = ?;
