-- OpenAPI 客户端表 (openapi_client)
-- 用于存储 OpenAPI 调用方的认证信息和配置
CREATE TABLE openapi_client (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    uuid VARCHAR(36) NOT NULL COMMENT 'UUID唯一标识',
    client_name VARCHAR(128) NOT NULL COMMENT '客户端名称',
    client_token VARCHAR(256) NOT NULL COMMENT '客户端Token',
    description VARCHAR(512) DEFAULT '' COMMENT '描述',
    owner VARCHAR(100) NOT NULL COMMENT '所有者(用户名)',
    status TINYINT DEFAULT 1 COMMENT '状态: 0-禁用, 1-启用',
    rate_limit INT DEFAULT 1000 COMMENT '每分钟请求限制',
    last_used_at DATETIME DEFAULT NULL COMMENT '最后使用时间',
    expires_at DATETIME DEFAULT NULL COMMENT '过期时间(NULL表示永不过期)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_uuid(uuid),
    UNIQUE KEY uk_client_token(client_token),
    INDEX idx_owner(owner),
    INDEX idx_status(status),
    INDEX idx_deleted_at(deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='OpenAPI客户端表';
