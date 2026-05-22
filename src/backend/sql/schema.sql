-- PromptOS Database Schema
-- Phase 1 MVP

CREATE DATABASE IF NOT EXISTS promptos DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE promptos;

-- Users table
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(20) NOT NULL UNIQUE,
    avatar VARCHAR(500) DEFAULT '',
    email VARCHAR(100) NOT NULL UNIQUE,
    github_id BIGINT NULL COMMENT 'GitHub user id',
    password VARCHAR(100) NULL DEFAULT NULL,
    bio VARCHAR(500) DEFAULT '',
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

-- Default categories
INSERT INTO categories (name, icon, sort, type) VALUES
('Image Generation', 'image', 1, 1),
('Copywriting', 'edit', 2, 1),
('Coding', 'code', 3, 1),
('Video Generation', 'video', 4, 1),
('Agent Prompt', 'robot', 5, 1),
('Workflow', 'workflow', 6, 1),
('Content Creation', 'content', 1, 2),
('Ecommerce Ops', 'shopping', 2, 2),
('Data Analysis', 'chart', 3, 2),
('AI Support', 'service', 4, 2),
('Coding Automation', 'auto', 5, 2),
('Multi-Agent Collaboration', 'multi-agent', 6, 2);
