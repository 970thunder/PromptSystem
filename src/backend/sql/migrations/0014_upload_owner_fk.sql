-- Enforce that upload metadata always belongs to a real user. This prevents
-- orphaned ownership records when callers bypass the application layer.
ALTER TABLE uploads
    ADD CONSTRAINT fk_uploads_owner
    FOREIGN KEY (owner_id) REFERENCES users(id)
    ON DELETE CASCADE;
