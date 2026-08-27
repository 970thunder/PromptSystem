-- Purpose: add users.session_version to revoke all JWT sessions on password
-- reset. Each password change increments the version; withAuth compares the
-- version embedded in the token against the current value, so old tokens for
-- that user immediately stop working without a central denylist scan.
-- Compatibility: idempotent. schema.sql includes this column on fresh installs,
-- so only add it when missing (avoids duplicate-column errors on existing DBs).
USE promptos;

SET @has_session_version := (
    SELECT COUNT(*)
    FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'users'
      AND COLUMN_NAME = 'session_version'
);

SET @add_session_version := IF(
    @has_session_version = 0,
    'ALTER TABLE users ADD COLUMN session_version INT NOT NULL DEFAULT 0 COMMENT ''Incremented to revoke all JWT sessions'' AFTER experience',
    'SELECT 1'
);
PREPARE stmt_session_version FROM @add_session_version;
EXECUTE stmt_session_version;
DEALLOCATE PREPARE stmt_session_version;
