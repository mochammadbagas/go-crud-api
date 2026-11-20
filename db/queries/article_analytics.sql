-- name: RecordAnalyticsEvent :exec
INSERT IGNORE INTO article_analytics_events (
    article_id, event_id, visitor_id, session_id,
    event_type, reading_time_seconds, scroll_depth, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetArticleDailyAnalytics :many
SELECT 
    id, article_id, analytics_date, total_views, unique_visitors,
    total_shares, total_reading_time, average_scroll_depth,
    created_at, updated_at
FROM article_daily_analytics
WHERE article_id = ? AND analytics_date >= ? AND analytics_date <= ?
ORDER BY analytics_date ASC;

-- name: AggregateDailyAnalyticsForDate :exec
INSERT INTO article_daily_analytics (
    article_id, analytics_date, total_views, unique_visitors,
    total_shares, total_reading_time, average_scroll_depth,
    created_at, updated_at
)
SELECT 
    e.article_id,
    ? AS analytics_date,
    COUNT(CASE WHEN e.event_type = 'view' THEN 1 END) AS total_views,
    COUNT(DISTINCT e.visitor_id) AS unique_visitors,
    COUNT(CASE WHEN e.event_type = 'share' THEN 1 END) AS total_shares,
    COALESCE(SUM(e.reading_time_seconds), 0) AS total_reading_time,
    COALESCE(AVG(CASE WHEN e.event_type = 'scroll' AND e.scroll_depth > 0 THEN e.scroll_depth END), 0.00) AS average_scroll_depth,
    NOW(),
    NOW()
FROM article_analytics_events e
WHERE e.created_at >= ? AND e.created_at < ?
GROUP BY e.article_id
ON DUPLICATE KEY UPDATE
    total_views = VALUES(total_views),
    unique_visitors = VALUES(unique_visitors),
    total_shares = VALUES(total_shares),
    total_reading_time = VALUES(total_reading_time),
    average_scroll_depth = VALUES(average_scroll_depth),
    updated_at = NOW();

-- name: PruneAnalyticsEvents :exec
DELETE FROM article_analytics_events
WHERE created_at < ?;

-- name: GetTotalAnalyticsOverview :one
SELECT 
    COALESCE(SUM(total_views), 0) AS global_views,
    COALESCE(SUM(unique_visitors), 0) AS global_unique_visitors,
    COALESCE(SUM(total_shares), 0) AS global_shares,
    COALESCE(SUM(total_reading_time), 0) AS global_reading_time
FROM article_daily_analytics;

-- name: GetUniqueVisitorsByDateRange :one
SELECT COUNT(DISTINCT visitor_id) AS unique_visitors
FROM article_analytics_events
WHERE article_id = ? AND created_at >= ? AND created_at <= ?;
