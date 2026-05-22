-- OAuth profile: GitHub usernames up to 39 chars; optional avatar/bio for OAuth-only users
USE promptos;

ALTER TABLE users
    MODIFY COLUMN username VARCHAR(39) NOT NULL;

ALTER TABLE users
    MODIFY COLUMN avatar VARCHAR(500) NULL DEFAULT NULL;

ALTER TABLE users
    MODIFY COLUMN bio VARCHAR(500) NULL DEFAULT NULL;
