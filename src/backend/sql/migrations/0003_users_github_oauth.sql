-- GitHub OAuth: github_id on users, nullable password for OAuth-only accounts
USE promptos;

ALTER TABLE users
    ADD COLUMN github_id BIGINT NULL COMMENT 'GitHub user id' AFTER email,
    ADD UNIQUE INDEX idx_github_id (github_id);

ALTER TABLE users
    MODIFY COLUMN password VARCHAR(100) NULL DEFAULT NULL;
