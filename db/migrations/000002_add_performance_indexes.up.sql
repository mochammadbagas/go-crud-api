-- Migration 000002: Add performance indexes for background worker pruning and public article queries
-- Up migration

CREATE INDEX idx_analytics_created_at ON article_analytics_events(created_at);
CREATE INDEX idx_articles_status_published ON articles(status, deleted_at, published_at);
