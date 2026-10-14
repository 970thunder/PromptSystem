-- Keyword search previously ran leading-wildcard LIKE across the large text
-- columns (title/description/content/system_prompt), forcing a full table
-- scan per search. The ngram FULLTEXT index enables CJK-aware phrase matching
-- (search correctness is preserved by pairing it with a light scan of the
-- small columns in the application query). Additive and idempotent.
USE promptos;

SET @has_ft := (
    SELECT COUNT(*) FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'prompts' AND INDEX_NAME = 'ft_prompts_text'
);
SET @add_ft := IF(
    @has_ft = 0,
    'ALTER TABLE prompts ADD FULLTEXT INDEX ft_prompts_text (title, description, content, system_prompt) WITH PARSER ngram',
    'SELECT 1'
);
PREPARE stmt_add_ft FROM @add_ft;
EXECUTE stmt_add_ft;
DEALLOCATE PREPARE stmt_add_ft;
