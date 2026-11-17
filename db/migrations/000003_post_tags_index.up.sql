-- Migration: 000003_post_tags_index.up.sql
-- Description: Add multi-valued index for JSON tags to optimize JSON_CONTAINS filtering in MySQL 8.0+

CREATE INDEX idx_posts_tags ON posts ( (CAST(tags AS CHAR(50) ARRAY)) );
