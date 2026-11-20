-- Migration 000002: Add performance indexes
-- Down migration

DROP INDEX idx_analytics_created_at ON article_analytics_events;
DROP INDEX idx_articles_status_published ON articles;
