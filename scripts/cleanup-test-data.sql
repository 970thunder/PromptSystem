-- ============================================================================
-- cleanup-test-data.sql — 清理集成测试残留在开发/生产库的垃圾数据
--
-- 背景：集成测试曾直接对运行库写入 "counter audit prompt *"、
--   "comment sort prompt *"、"MySQL interaction test" 等提示词及其测试用户，
--   它们会出现在首页「今日精选」、发现页首屏与 hero 飘带（旧版本）中。
--
-- ⚠️ 本脚本【不会】被任何启动流程自动执行。必须人工审核后手动运行。
-- ⚠️ 生产库执行前：先备份（mysqldump），先跑每个 SELECT 核对命中行。
-- ⚠️ 若集成测试未来仍直连运行库，根治办法是给测试使用独立库名
--   （PROMPTOS_TEST_MYSQL_DSN 指向 promptos_test），并重跑清理。
--
-- 使用方式（在 src/backend 环境变量同库下）：
--   mysql -h 127.0.0.1 -P 28303 -u root -p promptos < scripts/cleanup-test-data.sql
--   （生产端口与凭据见服务器部署说明；建议分步执行而非一次性导入）
-- ============================================================================

-- ---------- 第 1 步：预览将被删除的测试用户与内容（只读，先核对） ----------

-- 测试用户特征：集成测试生成的用户名前缀 / 固定名称
SELECT id, username, email, created_at FROM users
WHERE username LIKE 'counter\_%'
   OR username LIKE 'comment\_sort\_%'
   OR username LIKE 'it\_mysql\_%'
   OR username LIKE 'it\_%\_user';

-- 这些用户名下的提示词
SELECT p.id, p.title, p.user_id, p.status, p.created_at
FROM prompts p
JOIN users u ON u.id = p.user_id
WHERE u.username LIKE 'counter\_%'
   OR u.username LIKE 'comment\_sort\_%'
   OR u.username LIKE 'it\_mysql\_%'
   OR u.username LIKE 'it\_%\_user'
   OR p.title LIKE 'counter audit prompt %'
   OR p.title LIKE 'comment sort prompt %'
   OR p.title = 'MySQL interaction test';

-- ---------- 第 2 步：删除互动数据（依赖顺序：先子表再主表） ----------
-- 以下 DELETE 与第 1 步 SELECT 使用完全相同的范围条件；执行前务必先跑 SELECT。

SET @test_user_ids = NULL;
-- 用临时表圈定测试用户，避免每条 SQL 重复长条件
CREATE TEMPORARY TABLE tmp_test_users AS
SELECT id FROM users
WHERE username LIKE 'counter\_%'
   OR username LIKE 'comment\_sort\_%'
   OR username LIKE 'it\_mysql\_%'
   OR username LIKE 'it\_%\_user';

CREATE TEMPORARY TABLE tmp_test_prompts AS
SELECT p.id FROM prompts p
WHERE p.user_id IN (SELECT id FROM tmp_test_users)
   OR p.title LIKE 'counter audit prompt %'
   OR p.title LIKE 'comment sort prompt %'
   OR p.title = 'MySQL interaction test';

-- 评论（含评论上的点赞通过 target_type 关联）
DELETE FROM comments
WHERE (target_type = 'prompt' AND target_id IN (SELECT id FROM tmp_test_prompts))
   OR user_id IN (SELECT id FROM tmp_test_users);

DELETE FROM likes
WHERE (target_type = 'prompt' AND target_id IN (SELECT id FROM tmp_test_prompts))
   OR (target_type = 'comment' AND target_id NOT IN (SELECT id FROM comments))
   OR user_id IN (SELECT id FROM tmp_test_users);

DELETE FROM favorites
WHERE (target_type = 'prompt' AND target_id IN (SELECT id FROM tmp_test_prompts))
   OR user_id IN (SELECT id FROM tmp_test_users);

DELETE FROM view_histories
WHERE prompt_id IN (SELECT id FROM tmp_test_prompts)
   OR user_id IN (SELECT id FROM tmp_test_users);

DELETE FROM reports
WHERE (target_type = 'prompt' AND target_id IN (SELECT id FROM tmp_test_prompts))
   OR reporter_id IN (SELECT id FROM tmp_test_users);

-- 测试用户的上传记录（uploads 生命周期表；文件本体在对象存储/磁盘，可人工清理）
DELETE FROM uploads
WHERE user_id IN (SELECT id FROM tmp_test_users);

-- 提示词本体
DELETE FROM prompts WHERE id IN (SELECT id FROM tmp_test_prompts);

-- 最后删除测试用户
DELETE FROM users WHERE id IN (SELECT id FROM tmp_test_users);

DROP TEMPORARY TABLE tmp_test_prompts;
DROP TEMPORARY TABLE tmp_test_users;

-- ---------- 第 3 步：复核 ----------
-- SELECT COUNT(*) FROM prompts WHERE title LIKE 'counter audit prompt %';  -- 期望 0
-- SELECT COUNT(*) FROM users  WHERE username LIKE 'it\_%\_user';           -- 期望 0
