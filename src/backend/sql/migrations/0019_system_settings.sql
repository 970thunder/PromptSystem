-- Runtime site configuration for the first-run setup wizard and the admin
-- settings console. Values written here override environment defaults for the
-- settings the wizard owns (site identity, SMTP, registration toggle).
-- Secrets (smtp.password) must never be returned by public read APIs.
USE promptos;

CREATE TABLE IF NOT EXISTS system_settings (
    setting_key VARCHAR(64) PRIMARY KEY,
    setting_value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
