-- 创建数据库（首次部署时执行）
CREATE DATABASE IF NOT EXISTS content_creator
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

-- 使用数据库（后续表由 GORM AutoMigrate 自动创建）
USE content_creator;
