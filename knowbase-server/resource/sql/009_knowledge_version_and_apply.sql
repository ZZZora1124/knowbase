-- ============================================================================
-- 知识版本管理与应用功能 - 数据库迁移脚本
-- 版本: 007
-- 日期: 2026-03-16
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. 修改 knowledge 表，添加当前版本号字段
-- ----------------------------------------------------------------------------
ALTER TABLE knowledge ADD COLUMN current_version VARCHAR(50) NOT NULL DEFAULT 'v1' COMMENT '当前版本号';

-- 为现有数据设置默认版本
UPDATE knowledge SET current_version = 'v1' WHERE current_version = '' OR current_version IS NULL;

-- Knowledge表添加template_name字段
-- 用于标识知识是基于哪个模板创建的，默认为空
ALTER TABLE knowledge ADD COLUMN template_name VARCHAR(20) DEFAULT '' COMMENT '模板名称，标识知识基于哪个模板创建' AFTER knowledge_type;

-- ----------------------------------------------------------------------------
-- 2. 创建知识版本表 knowledge_version
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS knowledge_version (
                                                 id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                                 uuid VARCHAR(36) NOT NULL COMMENT '版本UUID',
    knowledge_uuid VARCHAR(36) NOT NULL COMMENT '所属知识UUID',
    version VARCHAR(50) NOT NULL COMMENT '版本号(支持自定义,如v1、v1.0.0、2026.03.12)',
    title VARCHAR(500) NOT NULL COMMENT '版本标题',
    content TEXT COMMENT '版本内容',
    file_path VARCHAR(1024) COMMENT '版本文件路径',
    file_size BIGINT DEFAULT 0 COMMENT '文件大小(字节)',
    file_format VARCHAR(50) COMMENT '文件格式',
    metadata JSON COMMENT '版本元数据',
    operator VARCHAR(100) NOT NULL COMMENT '操作人',
    operation ENUM('create', 'update', 'rollback') NOT NULL COMMENT '操作类型',
    remark VARCHAR(500) COMMENT '版本备注',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '版本创建时间',
    UNIQUE KEY uk_uuid(uuid),
    UNIQUE KEY uk_knowledge_version(knowledge_uuid, version),
    INDEX idx_knowledge_uuid(knowledge_uuid),
    INDEX idx_created_at(created_at),
    FOREIGN KEY (knowledge_uuid) REFERENCES knowledge(uuid) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识版本表';

-- ----------------------------------------------------------------------------
-- 3. 创建知识应用配置表 knowledge_apply_config
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS knowledge_apply_config (
                                                      id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                                      uuid VARCHAR(36) NOT NULL COMMENT '配置UUID',
    knowledge_base_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
    agent_code VARCHAR(100) NOT NULL COMMENT 'Agent标识码',
    agent_name VARCHAR(200) COMMENT 'Agent名称',
    scene VARCHAR(100) NOT NULL COMMENT '应用场景(如:code_review,coding)',
    target_type VARCHAR(100) NOT NULL COMMENT '应用对象类型(如:code_repo,card,project)',
    target_id VARCHAR(255) NOT NULL COMMENT '应用对象标识(如:仓库路径,卡片ID)',
    target_name VARCHAR(500) COMMENT '应用对象名称',
    target_extra JSON COMMENT '目标额外信息(JSON格式,如代码库的分支等)',
    version VARCHAR(50) NOT NULL DEFAULT 'v1' COMMENT '配置版本号(每次修改递增)',
    status ENUM('active', 'disabled') DEFAULT 'active' COMMENT '状态:active-生效,disabled-禁用',
    operator VARCHAR(100) NOT NULL COMMENT '最后操作人',
    remark VARCHAR(500) COMMENT '配置备注',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME COMMENT '删除时间',
    UNIQUE KEY uk_uuid(uuid),
    UNIQUE KEY uk_config(knowledge_base_uuid, agent_code, scene, target_type, target_id),
    INDEX idx_agent_scene(agent_code, scene),
    INDEX idx_target_type_id(target_type, target_id),
    INDEX idx_status(status),
    INDEX idx_operator(operator),
    INDEX idx_created_at(created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识应用配置表(按目标维度聚合)';

-- ----------------------------------------------------------------------------
-- 4. 创建知识应用记录表 knowledge_apply_record
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS knowledge_apply_record (
                                                      id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                                      uuid VARCHAR(36) NOT NULL COMMENT '记录UUID',
    config_uuid VARCHAR(36) NOT NULL COMMENT '所属应用配置UUID',
    config_version VARCHAR(50) NOT NULL COMMENT '应用时的配置版本号',
    knowledge_base_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
    knowledge_uuid VARCHAR(36) NOT NULL COMMENT '知识UUID',
    knowledge_version VARCHAR(50) NOT NULL COMMENT '应用的知识版本号',
    operator VARCHAR(100) NOT NULL COMMENT '应用人',
    push_status ENUM('pending', 'success', 'failed') DEFAULT 'pending' COMMENT '推送状态',
    push_error_msg TEXT COMMENT '推送失败原因',
    pushed_at DATETIME COMMENT '推送时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_uuid(uuid),
    INDEX idx_config_version(config_uuid, config_version),
    INDEX idx_knowledge_base_uuid(knowledge_base_uuid),
    INDEX idx_knowledge_uuid(knowledge_uuid),
    INDEX idx_push_status(push_status),
    INDEX idx_created_at(created_at),
    FOREIGN KEY (config_uuid) REFERENCES knowledge_apply_config(uuid) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识应用记录表(明细)';

-- ----------------------------------------------------------------------------
-- 5. 操作日志增加agent字段，用于记录哪个agent使用用户身份操作了知识
-- ----------------------------------------------------------------------------
-- 新增 agent 字段
ALTER TABLE knowledgebase_operation_log ADD COLUMN agent VARCHAR(32) DEFAULT NULL COMMENT '操作Agent标识' AFTER operator;

-- 为 agent 字段添加索引
ALTER TABLE knowledgebase_operation_log ADD INDEX idx_agent(agent);

-- ----------------------------------------------------------------------------
-- 6. openapi_client增加isAdmin字段，用于区分是否为Admin权限
-- ----------------------------------------------------------------------------
ALTER TABLE openapi_client ADD COLUMN is_admin TINYINT DEFAULT 0 COMMENT 'Admin权限标识: 0-否, 1-是' AFTER status;

-- ----------------------------------------------------------------------------
-- 7. 数据迁移：为现有知识创建v1版本记录
-- ----------------------------------------------------------------------------
INSERT INTO knowledge_version (
    uuid, knowledge_uuid, version, title, content,
    file_path, file_size, file_format, metadata,
    operator, operation, created_at
)
SELECT
    k.uuid as uuid,
    k.uuid as knowledge_uuid,
    'v1' as version,
    k.title,
    k.content,
    k.file_path,
    k.file_size,
    k.file_format,
    k.metadata,
    k.creator_username as operator,
    'create' as operation,
    k.created_at
FROM knowledge k
WHERE k.deleted_at IS NULL
  AND NOT EXISTS (
        SELECT 1 FROM knowledge_version kv
        WHERE kv.knowledge_uuid = k.uuid
    );
