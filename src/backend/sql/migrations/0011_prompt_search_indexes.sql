-- Purpose: add the composite indexes that the PromptOS search and listing
-- queries rely on, so MySQL serves prompt list/search/tag queries without full
-- table scans as the table grows. MySQL search is kept (no ElasticSearch is
-- introduced); keyword matching uses bounded LIKE predicates, so these indexes
-- target exact-match and range filters (status, category, user, interaction
-- counters) plus tag lookups.
--
-- Indexes and their purpose:
--   idx_status_category_created (status, category_id, created_at)
--     Serves "recent published prompts in a category" (filter by status +
--     category, order by created_at) and the home "recent" listing.
--   idx_user_created (user_id, created_at)
--     Serves "someone's published prompts ordered by recency".
--   idx_status_popular (status, likes)
--     Serves the "popular" sort (status = 1 filtered by likes) and the
--     interaction-sort path; id included implicitly via the covering of likes.
--   idx_prompt_tag (prompt_tags: prompt_id, tag)
--     Serves tag-based EXISTS lookups (filter_tags.prompt_id = p.id AND tag = ?)
--     and tag-after-tag filtering; complements the unique index added in 0010.
--
-- Compatibility: idempotent. Each index is added only when not already present,
-- so replaying on an existing baseline (schema.sql already enumerates them) is
-- safe. No index is dropped here.
--
-- Irreversibility: dropping the indexes only trades query performance for DML
-- write overhead; removing them does not corrupt data. They are retained because
-- the listed queries need them at production scale.
USE promptos;

-- status + category + created ordering.
SET @has_status_category_created := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompts'
      AND INDEX_NAME = 'idx_status_category_created'
);
SET @ddl := IF(
    @has_status_category_created = 0,
    'ALTER TABLE prompts ADD INDEX idx_status_category_created (status, category_id, created_at)',
    'SELECT 1'
);
PREPARE stmt_status_category_created FROM @ddl;
EXECUTE stmt_status_category_created;
DEALLOCATE PREPARE stmt_status_category_created;

-- user + created ordering.
SET @has_user_created := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompts'
      AND INDEX_NAME = 'idx_user_created'
);
SET @ddl := IF(
    @has_user_created = 0,
    'ALTER TABLE prompts ADD INDEX idx_user_created (user_id, created_at)',
    'SELECT 1'
);
PREPARE stmt_user_created FROM @ddl;
EXECUTE stmt_user_created;
DEALLOCATE PREPARE stmt_user_created;

-- interaction (popular) sort with a status filter.
SET @has_status_popular := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompts'
      AND INDEX_NAME = 'idx_status_popular'
);
SET @ddl := IF(
    @has_status_popular = 0,
    'ALTER TABLE prompts ADD INDEX idx_status_popular (status, likes)',
    'SELECT 1'
);
PREPARE stmt_status_popular FROM @ddl;
EXECUTE stmt_status_popular;
DEALLOCATE PREPARE stmt_status_popular;

-- tag search: prompt_id + tag for the EXISTS tag filter.
SET @has_prompt_tag := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompt_tags'
      AND INDEX_NAME = 'idx_prompt_tag'
);
SET @ddl := IF(
    @has_prompt_tag = 0,
    'ALTER TABLE prompt_tags ADD INDEX idx_prompt_tag (prompt_id, tag)',
    'SELECT 1'
);
PREPARE stmt_prompt_tag FROM @ddl;
EXECUTE stmt_prompt_tag;
DEALLOCATE PREPARE stmt_prompt_tag;
