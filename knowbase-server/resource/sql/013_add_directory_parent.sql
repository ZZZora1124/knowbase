-- 目录表新增 parent_uuid 字段，支持多级目录（邻接表方案）
-- Date: 2026-04-22
-- Description: parent_uuid=NULL 表示根目录（一级目录）

ALTER TABLE knowledgebase_directory
  ADD COLUMN parent_uuid VARCHAR(36) DEFAULT NULL
    COMMENT '父目录UUID，NULL表示根目录（一级目录）'
    AFTER knowledgebase_uuid,
  ADD INDEX idx_parent_uuid(parent_uuid);
