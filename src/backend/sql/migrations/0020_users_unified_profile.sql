-- 统一资料（昵称、头像）由 isoumao 身份中心权威维护；本站只保存一份用于展示的副本。
ALTER TABLE users
    ADD COLUMN display_name VARCHAR(64) NULL COMMENT '统一账号昵称（身份中心同步）' AFTER username;

ALTER TABLE users
    ADD COLUMN avatar_url VARCHAR(500) NULL COMMENT '统一账号头像地址（身份中心同步）' AFTER avatar;

ALTER TABLE users
    ADD COLUMN profile_synced_at DATETIME NULL COMMENT '最近一次统一资料同步时间' AFTER avatar_url;
