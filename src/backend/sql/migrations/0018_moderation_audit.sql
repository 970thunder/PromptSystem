-- Add administrator roles, review metadata and a tamper-evident audit chain.
-- All statements are additive so this migration is safe on existing releases.
USE promptos;

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT PRIMARY KEY,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_roles_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @has_reviewed_by := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'reports' AND COLUMN_NAME = 'reviewed_by'
);
SET @add_reviewed_by := IF(
    @has_reviewed_by = 0,
    'ALTER TABLE reports ADD COLUMN reviewed_by BIGINT NULL AFTER status',
    'SELECT 1'
);
PREPARE stmt_reviewed_by FROM @add_reviewed_by;
EXECUTE stmt_reviewed_by;
DEALLOCATE PREPARE stmt_reviewed_by;

SET @has_review_note := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'reports' AND COLUMN_NAME = 'review_note'
);
SET @add_review_note := IF(
    @has_review_note = 0,
    'ALTER TABLE reports ADD COLUMN review_note VARCHAR(500) NOT NULL DEFAULT '''' AFTER reviewed_by',
    'SELECT 1'
);
PREPARE stmt_review_note FROM @add_review_note;
EXECUTE stmt_review_note;
DEALLOCATE PREPARE stmt_review_note;

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    actor_user_id BIGINT NOT NULL,
    action VARCHAR(80) NOT NULL,
    target_type VARCHAR(32) NOT NULL,
    target_id BIGINT NOT NULL,
    metadata JSON NOT NULL,
    request_id VARCHAR(128) DEFAULT '',
    prev_hash CHAR(64) NOT NULL DEFAULT '',
    event_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (actor_user_id) REFERENCES users(id),
    INDEX idx_audit_created (created_at, id),
    INDEX idx_audit_target (target_type, target_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
