-- Purpose: fix seed text encoding for existing demo rows.
-- Compatibility: safe on both fresh and existing databases. This migration only
-- updates rows that already exist; it never inserts prompt_tags before the
-- matching prompts exist (that FK dependency used to break fresh installs).
-- Demo seed content is maintained idempotently in store/mysql_seed.go instead.

UPDATE categories SET name = '摄影' WHERE id = 1;
UPDATE categories SET name = '插画' WHERE id = 2;
UPDATE categories SET name = '3D' WHERE id = 3;
UPDATE categories SET name = '电商' WHERE id = 4;
UPDATE categories SET name = '人像' WHERE id = 5;
UPDATE categories SET name = '建筑' WHERE id = 6;
UPDATE categories SET name = '动漫' WHERE id = 7;
UPDATE categories SET name = 'UI' WHERE id = 8;
UPDATE categories SET name = '海报' WHERE id = 9;
UPDATE categories SET name = '产品' WHERE id = 10;
UPDATE categories SET name = '风景' WHERE id = 11;
UPDATE categories SET name = '美食' WHERE id = 12;
