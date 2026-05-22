-- GitHub OAuth: github_id on users, nullable password for OAuth-only accounts
USE promptos;

SET @has_github_id := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'users'
      AND COLUMN_NAME = 'github_id'
);

SET @add_github_id := IF(
    @has_github_id = 0,
    'ALTER TABLE users ADD COLUMN github_id BIGINT NULL COMMENT ''GitHub user id'' AFTER email',
    'SELECT 1'
);
PREPARE stmt_github_col FROM @add_github_id;
EXECUTE stmt_github_col;
DEALLOCATE PREPARE stmt_github_col;

SET @has_github_idx := (
    SELECT COUNT(*)
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'users'
      AND INDEX_NAME = 'idx_github_id'
);

SET @add_github_idx := IF(
    @has_github_idx = 0,
    'CREATE UNIQUE INDEX idx_github_id ON users (github_id)',
    'SELECT 1'
);
PREPARE stmt_github_idx FROM @add_github_idx;
EXECUTE stmt_github_idx;
DEALLOCATE PREPARE stmt_github_idx;

ALTER TABLE users
    MODIFY COLUMN password VARCHAR(100) NULL DEFAULT NULL;
