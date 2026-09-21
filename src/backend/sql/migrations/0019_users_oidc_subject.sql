ALTER TABLE users
    ADD COLUMN oidc_subject VARCHAR(255) NULL COMMENT 'Keycloak OIDC subject';

ALTER TABLE users
    ADD UNIQUE INDEX idx_oidc_subject (oidc_subject);
