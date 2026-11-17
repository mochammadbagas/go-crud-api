-- Migration: 000003_post_tags_index.down.sql
-- Description: Drop multi-valued index for JSON tags

DROP INDEX idx_posts_tags ON posts;
