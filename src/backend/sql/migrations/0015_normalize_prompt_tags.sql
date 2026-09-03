-- Purpose: normalize existing prompt tags to the same canonical form used by
-- the store (trim, collapse whitespace, lowercase) and remove collisions that
-- become duplicates after normalization.
--
-- Compatibility: idempotent. The duplicate cleanup runs before the update so
-- the existing unique index on (prompt_id, tag) cannot reject the normalized
-- values. MySQL 8 REGEXP_REPLACE handles all Unicode whitespace classes that
-- can occur in imported tags.
--
-- Irreversibility: original capitalization and spacing are intentionally not
-- retained; the canonical value is the application contract.
USE promptos;

DELETE duplicate_row
FROM prompt_tags duplicate_row
JOIN prompt_tags keep_row
  ON duplicate_row.prompt_id = keep_row.prompt_id
 AND duplicate_row.id > keep_row.id
 AND LOWER(TRIM(REGEXP_REPLACE(duplicate_row.tag, '[[:space:]]+', ' '))) =
     LOWER(TRIM(REGEXP_REPLACE(keep_row.tag, '[[:space:]]+', ' ')));

UPDATE prompt_tags
SET tag = LOWER(TRIM(REGEXP_REPLACE(tag, '[[:space:]]+', ' ')))
WHERE BINARY tag <> BINARY LOWER(TRIM(REGEXP_REPLACE(tag, '[[:space:]]+', ' ')));
