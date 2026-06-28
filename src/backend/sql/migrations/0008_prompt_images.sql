ALTER TABLE prompts ADD COLUMN images JSON NULL COMMENT 'Additional result image URLs' AFTER cover;
