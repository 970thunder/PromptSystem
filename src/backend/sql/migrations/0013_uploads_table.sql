-- Purpose: track every stored object so the backend can prove ownership of a
-- private temporary upload and garbage-collect unreferenced objects instead of
-- leaking them in the bucket. Each row records who uploaded it, which provider
-- stored the bytes, the stable object key, the content type and size, and a
-- lifecycle status (pending -> referenced -> trashed).
--
-- Object keys are derived from (userID, purpose, random) -- never from the raw
-- client filename -- so one user cannot guess or overwrite another user's
-- uploads, and collisions are impossible.
--
-- Lifecycle:
--   * On upload the row is inserted with status='pending'.
--   * When a prompt cover/image or avatar begins referencing the key, the row is
--     flipped to 'referenced' so cleanup skips it.
--   * A periodic cleanup transitions still-'pending' rows older than a threshold
--     to 'trashed' (soft delete) and the storage layer removes the bytes; hard
--     deletes are never done inline.
--
-- Compatibility / idempotency: additive table, safe to run on fresh and existing
-- databases. Re-running converges (the table already exists, the insert guard via
-- the unique key prevents duplicates). The statements are wrapped in a
-- create-if-not-exists guard so a re-applied migration is a no-op.
-- USE promptos;

CREATE TABLE IF NOT EXISTS uploads (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    owner_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL DEFAULT 'local',
    purpose VARCHAR(32) NOT NULL DEFAULT 'prompt_image',
    object_key VARCHAR(512) NOT NULL,
    content_type VARCHAR(128) NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending, referenced, trashed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_object_key (object_key),
    INDEX idx_owner (owner_id),
    INDEX idx_status_created (status, created_at),
    INDEX idx_purpose (purpose)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
