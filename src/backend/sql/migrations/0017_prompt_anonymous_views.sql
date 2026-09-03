-- Purpose: preserve the anonymous portion of prompts.views so counter audits
-- can distinguish anonymous views from the logged-in rows in view_histories.
-- Compatibility: idempotent; existing views are split into the known history
-- count plus a non-negative anonymous remainder. No view history is changed.
-- The column is intentionally maintained by the RecordView transaction.
USE promptos;

SET @has_anonymous_views := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompts'
      AND COLUMN_NAME = 'anonymous_views'
);
SET @ddl := IF(
    @has_anonymous_views = 0,
    'ALTER TABLE prompts ADD COLUMN anonymous_views INT NOT NULL DEFAULT 0 AFTER views',
    'SELECT 1'
);
PREPARE stmt_anonymous_views FROM @ddl;
EXECUTE stmt_anonymous_views;
DEALLOCATE PREPARE stmt_anonymous_views;

UPDATE prompts p
LEFT JOIN (
    SELECT prompt_id, COUNT(*) AS cnt
    FROM view_histories
    GROUP BY prompt_id
) h ON h.prompt_id = p.id
SET p.anonymous_views = GREATEST(p.views - COALESCE(h.cnt, 0), 0);
