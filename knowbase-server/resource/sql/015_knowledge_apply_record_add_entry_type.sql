-- ----------------------------------------------------------------------------
-- 知识应用记录表新增条目类型支持：目录 UUID 作为应用单位
-- ----------------------------------------------------------------------------
-- 背景：
--   原 knowledge_apply_record 表每条记录仅代表「应用某条知识」。
--   本次扩展支持「应用某个目录」（对应该目录及其所有后代目录下的知识，
--   在查询/推送时动态展开）。
-- 字段说明：
--   - entry_type: 记录条目类型
--       * knowledge (默认): 应用单条知识，使用 knowledge_uuid / knowledge_version
--       * directory         : 应用整个目录子树，使用 directory_uuid；
--                             knowledge_uuid / knowledge_version 置空串
--   - directory_uuid: 目录UUID，仅 entry_type='directory' 时有效
-- 兼容性：
--   存量数据由于 entry_type 默认值 'knowledge' 无需回填。
-- ----------------------------------------------------------------------------

ALTER TABLE knowledge_apply_record
    ADD COLUMN entry_type VARCHAR(20) NOT NULL DEFAULT 'knowledge'
        COMMENT '条目类型: knowledge(知识)/directory(目录)' AFTER knowledge_version,
    ADD COLUMN directory_uuid VARCHAR(36) NULL
        COMMENT '目录UUID(entry_type=directory时使用)' AFTER entry_type,
    ADD INDEX idx_entry_type (entry_type),
    ADD INDEX idx_directory_uuid (directory_uuid);
