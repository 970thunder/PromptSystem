-- PromptOS Database Schema
-- Phase 1 MVP

CREATE DATABASE IF NOT EXISTS promptos DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE promptos;

-- Users table
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(39) NOT NULL UNIQUE,
    avatar VARCHAR(500) NULL DEFAULT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    github_id BIGINT NULL COMMENT 'GitHub user id',
    password VARCHAR(100) NULL DEFAULT NULL,
    bio VARCHAR(500) NULL DEFAULT NULL,
    level INT DEFAULT 1,
    experience INT DEFAULT 0,
    status INT DEFAULT 1 COMMENT '1:active, 0:disabled',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_username (username),
    UNIQUE INDEX idx_github_id (github_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Categories table
CREATE TABLE categories (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    icon VARCHAR(100) DEFAULT '',
    sort INT DEFAULT 0,
    type INT DEFAULT 1 COMMENT '1:prompt, 2:skill',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Prompts table
CREATE TABLE prompts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    description VARCHAR(1000) DEFAULT '',
    cover VARCHAR(1024) DEFAULT '',
    content TEXT NOT NULL,
    system_prompt VARCHAR(2000) DEFAULT '',
    model VARCHAR(50) DEFAULT '',
    params JSON NULL COMMENT 'JSON format',
    category_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    views INT DEFAULT 0,
    likes INT DEFAULT 0,
    favorites INT DEFAULT 0,
    status INT DEFAULT 1 COMMENT '1:published, 0:draft, -1:deleted',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category_id),
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_created (created_at),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Skills table
CREATE TABLE skills (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000) DEFAULT '',
    cover VARCHAR(500) DEFAULT '',
    workflow JSON NOT NULL COMMENT 'Workflow steps as JSON',
    io_schema JSON NOT NULL COMMENT 'Input/output schema as JSON',
    user_id BIGINT NOT NULL,
    views INT DEFAULT 0,
    likes INT DEFAULT 0,
    favorites INT DEFAULT 0,
    status INT DEFAULT 1 COMMENT '1:published, 0:draft, -1:deleted',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Prompt tags table
CREATE TABLE prompt_tags (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    prompt_id BIGINT NOT NULL,
    tag VARCHAR(50) NOT NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE,
    INDEX idx_prompt (prompt_id),
    INDEX idx_tag (tag)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Comments table
CREATE TABLE comments (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    target_type VARCHAR(20) NOT NULL COMMENT 'prompt, skill',
    target_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    content VARCHAR(1000) NOT NULL,
    parent_id BIGINT DEFAULT NULL,
    likes INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_target (target_type, target_id),
    INDEX idx_user (user_id),
    INDEX idx_parent (parent_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Favorites table
CREATE TABLE favorites (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL COMMENT 'prompt, skill',
    target_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_target (user_id, target_type, target_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Likes table
CREATE TABLE likes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL COMMENT 'prompt, skill, comment',
    target_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_target (user_id, target_type, target_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Reports table
CREATE TABLE reports (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    target_type VARCHAR(20) NOT NULL COMMENT 'comment, prompt, skill',
    target_id BIGINT NOT NULL,
    reason VARCHAR(80) NOT NULL,
    detail VARCHAR(500) DEFAULT '',
    status VARCHAR(20) DEFAULT 'pending' COMMENT 'pending, reviewed, rejected',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_target (user_id, target_type, target_id),
    INDEX idx_target (target_type, target_id),
    INDEX idx_status (status),
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- View histories table
CREATE TABLE view_histories (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    prompt_id BIGINT NOT NULL,
    viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_prompt (user_id, prompt_id),
    INDEX idx_user_viewed (user_id, viewed_at),
    INDEX idx_prompt (prompt_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Follows table
CREATE TABLE follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    follower_id BIGINT NOT NULL,
    following_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_follow (follower_id, following_id),
    FOREIGN KEY (follower_id) REFERENCES users(id),
    FOREIGN KEY (following_id) REFERENCES users(id),
    INDEX idx_follower (follower_id),
    INDEX idx_following (following_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Default categories (prompt: image-focused Chinese labels)
INSERT INTO categories (id, name, icon, sort, type) VALUES
(1, '摄影', 'camera', 1, 1),
(2, '插画', 'brush', 2, 1),
(3, '3D', 'cube', 3, 1),
(4, '电商', 'shopping', 4, 1),
(5, '人像', 'portrait', 5, 1),
(6, '建筑', 'building', 6, 1),
(7, '动漫', 'anime', 7, 1),
(8, 'UI', 'layout', 8, 1),
(9, '海报', 'poster', 9, 1),
(10, '产品', 'product', 10, 1),
(11, '风景', 'landscape', 11, 1),
(12, '美食', 'food', 12, 1),
(13, '时尚', 'fashion', 13, 1),
(14, '游戏', 'game', 14, 1),
(15, '图标', 'icon', 15, 1),
(16, 'LOGO', 'logo', 16, 1),
(17, '室内设计', 'interior', 17, 1),
(18, '汽车', 'car', 18, 1),
(19, '宠物', 'pet', 19, 1),
(20, '婚礼', 'wedding', 20, 1),
(21, '科幻', 'scifi', 21, 1),
(22, '水彩', 'watercolor', 22, 1),
(23, '油画', 'oil', 23, 1),
(24, '像素', 'pixel', 24, 1),
(25, '线稿', 'lineart', 25, 1),
(26, '表情包', 'emoji', 26, 1),
(27, '壁纸', 'wallpaper', 27, 1),
(28, '社交媒体', 'social', 28, 1),
(29, '广告创意', 'ads', 29, 1),
(30, '其他', 'more', 30, 1);

INSERT INTO categories (name, icon, sort, type) VALUES
('Content Creation', 'content', 1, 2),
('Ecommerce Ops', 'shopping', 2, 2),
('Data Analysis', 'chart', 3, 2),
('AI Support', 'service', 4, 2),
('Coding Automation', 'auto', 5, 2),
('Multi-Agent Collaboration', 'multi-agent', 6, 2);
