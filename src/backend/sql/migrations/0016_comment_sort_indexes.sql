-- Purpose: support paginated root-comment ordering after the sort is selected
-- in SQL. The target and parent columns narrow the range first; the created_at
-- index serves latest/oldest and the likes index serves popular.
--
-- Compatibility: idempotent. Existing baseline schemas already include these
-- indexes, while older deployments add each one exactly once.
USE promptos;

SET @has_target_parent_created := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'comments'
      AND INDEX_NAME = 'idx_target_parent_created'
);
SET @ddl := IF(
    @has_target_parent_created = 0,
    'ALTER TABLE comments ADD INDEX idx_target_parent_created (target_type, target_id, parent_id, created_at, id)',
    'SELECT 1'
);
PREPARE stmt_target_parent_created FROM @ddl;
EXECUTE stmt_target_parent_created;
DEALLOCATE PREPARE stmt_target_parent_created;

SET @has_target_parent_likes := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'comments'
      AND INDEX_NAME = 'idx_target_parent_likes'
);
SET @ddl := IF(
    @has_target_parent_likes = 0,
    'ALTER TABLE comments ADD INDEX idx_target_parent_likes (target_type, target_id, parent_id, likes, created_at, id)',
    'SELECT 1'
);
PREPARE stmt_target_parent_likes FROM @ddl;
EXECUTE stmt_target_parent_likes;
DEALLOCATE PREPARE stmt_target_parent_likes;
