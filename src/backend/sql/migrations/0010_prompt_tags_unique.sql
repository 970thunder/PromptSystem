-- Purpose: enforce that a tag appears at most once per prompt by adding a
-- UNIQUE index on prompt_tags(prompt_id, tag). Combined with store-side tag
-- normalization (trim, whitespace-collapse, dedupe), this guarantees consistent
-- tag storage and prevents duplicate capability/theme tags from accruing.
--
-- Compatibility: idempotent. It first deletes duplicate prompt_tags rows,
-- keeping the lowest id in each (prompt_id, tag) group, so existing databases
-- that already contain duplicates (e.g. produced before normalization existed)
-- can still apply this migration cleanly. The unique index is added only when it
-- is not already present.
--
-- Irreversibility: dropping the unique index is the only way to revert, but
-- doing so would re-allow duplicate tags. The normalization applied at the store
-- layer makes duplicates unreachable for new writes regardless.
USE promptos;

-- Remove any pre-existing duplicate tag rows before adding the unique index.
DELETE pt
FROM prompt_tags pt
JOIN prompt_tags duplicate_row
  ON pt.prompt_id = duplicate_row.prompt_id
 AND pt.tag = duplicate_row.tag
 AND pt.id > duplicate_row.id;

SET @has_unique := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompt_tags'
      AND INDEX_NAME = 'uk_prompt_tag'
);

SET @add_unique := IF(
    @has_unique = 0,
    'ALTER TABLE prompt_tags ADD UNIQUE INDEX uk_prompt_tag (prompt_id, tag)',
    'SELECT 1'
);
PREPARE stmt_prompt_tag_unique FROM @add_unique;
EXECUTE stmt_prompt_tag_unique;
DEALLOCATE PREPARE stmt_prompt_tag_unique;
