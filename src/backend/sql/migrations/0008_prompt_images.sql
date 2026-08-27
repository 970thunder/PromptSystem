-- Purpose: add the images JSON column to prompts for additional result images.
-- Compatibility: idempotent. schema.sql already includes this column on fresh
-- installs, so only add it when it is missing (avoids duplicate-column errors
-- when migrations run on top of the fresh baseline).
USE promptos;

SET @has_images := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'prompts'
      AND COLUMN_NAME = 'images'
);

SET @add_images := IF(
    @has_images = 0,
    'ALTER TABLE prompts ADD COLUMN images JSON NULL COMMENT ''Additional result image URLs'' AFTER cover',
    'SELECT 1'
);
PREPARE stmt_images_col FROM @add_images;
EXECUTE stmt_images_col;
DEALLOCATE PREPARE stmt_images_col;
