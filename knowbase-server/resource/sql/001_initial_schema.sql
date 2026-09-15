-- 知识本表 (knowledgebase)
CREATE TABLE knowledgebase (
                               id BIGINT PRIMARY KEY AUTO_INCREMENT,
                               uuid VARCHAR(36) NOT NULL COMMENT 'UUID唯一标识',
                               name VARCHAR(128) NOT NULL COMMENT '知识本名称,全局唯一,支持中英文、数字、中划线、下划线',
                               description VARCHAR(512) NOT NULL DEFAULT '' COMMENT '描述',
                               visibility ENUM('private', 'public', 'personal') DEFAULT 'private' COMMENT '可见性: private-私密, public-公开, personal-个人知识本(仅自己可见)',
                               owner VARCHAR(100) NOT NULL COMMENT '所有者(用户名)',
                               status TINYINT DEFAULT 1 COMMENT '状态: 0-禁用, 1-启用',
                               created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                               updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                               deleted_at DATETIME DEFAULT NULL COMMENT '软删除时间',
                               UNIQUE KEY uk_name(name),
                               UNIQUE KEY uk_uuid(uuid),
                               INDEX idx_owner(owner),
                               INDEX idx_created(created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识本表';

-- 目录表 (knowledgebase_directory)
CREATE TABLE knowledgebase_directory (
                                         id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                         uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                         knowledgebase_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
                                         name VARCHAR(255) NOT NULL COMMENT '目录名称,支持中英文、数字、中划线、下划线',
                                         created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                         updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                         deleted_at DATETIME DEFAULT NULL,
                                         UNIQUE KEY uk_uuid(uuid),
                                         INDEX idx_kb_uuid(knowledgebase_uuid),
                                         INDEX idx_created(created_at),
                                         FOREIGN KEY (knowledgebase_uuid) REFERENCES knowledgebase(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识本目录表(仅用于自定义知识,只支持一级目录)';

ALTER TABLE knowledgebase_directory
DROP INDEX idx_kb_uuid,
ADD INDEX idx_kb_uuid_name(knowledgebase_uuid, name);

-- 知识本成员表 (knowledgebase_member)
CREATE TABLE knowledgebase_member (
                                      id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                      uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                      knowledgebase_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
                                      member_type ENUM('username', 'email_group', 'department') NOT NULL COMMENT '成员类型: username-用户, email_group-邮件组, department-部门',
                                      member_value VARCHAR(255) NOT NULL COMMENT '成员值: 用户名/邮件组名/部门名',
                                      role ENUM('admin', 'editor', 'viewer') DEFAULT 'viewer' COMMENT '角色: admin-管理员, editor-编辑者, viewer-查看者',
                                      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                      updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                      UNIQUE KEY uk_kb_member(knowledgebase_uuid, member_type, member_value),
                                      UNIQUE KEY uk_uuid(uuid),
                                      INDEX idx_kb_uuid(knowledgebase_uuid),
                                      INDEX idx_created(created_at),
                                      FOREIGN KEY (knowledgebase_uuid) REFERENCES knowledgebase(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识本成员表';

-- 知识表 (knowledge)
CREATE TABLE knowledge (
                           id BIGINT PRIMARY KEY AUTO_INCREMENT,
                           uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                           knowledgebase_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
                           directory_uuid VARCHAR(36) DEFAULT NULL COMMENT '所属目录UUID(仅自定义知识使用)',
                           creator_username VARCHAR(100) NOT NULL COMMENT '创建者用户名',
                           title VARCHAR(500) NOT NULL COMMENT '知识标题',
                           knowledge_type ENUM('content', 'text_file', 'audio_file', 'image_file', 'video_file', 'config') NOT NULL COMMENT '知识类型',
                           knowledge_source ENUM('custom', 'icode', 'ku', 'icafe', 'other') DEFAULT 'custom' COMMENT '知识来源',
                           content TEXT COMMENT '短文本内容(knowledge_type=content时使用)',
                           file_path VARCHAR(1024) COMMENT 'BOS文件路径(文件类型时使用)',
                           file_size BIGINT DEFAULT 0 COMMENT '文件大小(字节)',
                           file_format VARCHAR(50) COMMENT '文件格式(如: pdf, mp3, jpg等)',
                           source_method VARCHAR(100) DEFAULT 'manual' COMMENT '收集方式: manual-手动上传, api-API上传, webhook-第三方集成',
                           metadata JSON COMMENT '扩展元数据,存储知识源特定信息(如iCode仓库URL、iCafe卡片ID等)',
                           created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                           updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                           deleted_at DATETIME DEFAULT NULL,
                           UNIQUE KEY uk_uuid(uuid),
                           INDEX idx_kb_uuid(knowledgebase_uuid),
                           INDEX idx_dir_uuid(directory_uuid),
                           INDEX idx_creator(creator_username),
                           INDEX idx_type(knowledge_type),
                           INDEX idx_source(knowledge_source),
                           INDEX idx_method(source_method),
                           INDEX idx_created(created_at),
                           FOREIGN KEY (knowledgebase_uuid) REFERENCES knowledgebase(uuid) ON DELETE CASCADE,
                           FOREIGN KEY (directory_uuid) REFERENCES knowledgebase_directory(uuid) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识表';

-- 标签表 (knowledgebase_tag)
CREATE TABLE knowledgebase_tag (
                                   id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                   uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                   knowledgebase_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
                                   name VARCHAR(100) NOT NULL COMMENT '标签名称',
                                   description VARCHAR(500) COMMENT '标签描述',
                                   created_by VARCHAR(100) NOT NULL COMMENT '创建者用户名',
                                   created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                   updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                   deleted_at DATETIME DEFAULT NULL,
                                   UNIQUE KEY uk_kb_name(knowledgebase_uuid, name),
                                   UNIQUE KEY uk_uuid(uuid),
                                   INDEX idx_created(created_at),
                                   FOREIGN KEY (knowledgebase_uuid) REFERENCES knowledgebase(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识本标签表';

-- 知识-标签关联表 (knowledge_tag_relation)
CREATE TABLE knowledge_tag_relation (
                                        id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                        uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                        knowledge_uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                        tag_uuid VARCHAR(36) NOT NULL COMMENT 'UUID',
                                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                        UNIQUE KEY uk_knowledge_tag(knowledge_uuid, tag_uuid),
                                        INDEX idx_tag(tag_uuid),
                                        INDEX idx_created(created_at),
                                        FOREIGN KEY (knowledge_uuid) REFERENCES knowledge(uuid) ON DELETE CASCADE,
                                        FOREIGN KEY (tag_uuid) REFERENCES knowledgebase_tag(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识标签关联表';

-- 知识本操作记录表 (knowledgebase_operation_log)
CREATE TABLE knowledgebase_operation_log (
                                             id BIGINT PRIMARY KEY AUTO_INCREMENT,
                                             knowledgebase_uuid VARCHAR(36) NOT NULL COMMENT '所属知识本UUID',
                                             operator VARCHAR(100) NOT NULL COMMENT '操作者用户名',
                                             operation VARCHAR(100) NOT NULL COMMENT '操作类型: create_kb/update_kb/delete_kb/create_dir/delete_dir/create_knowledge/update_knowledge/delete_knowledge/add_member/remove_member/create_tag/delete_tag等',
                                             resource_type VARCHAR(50) NOT NULL COMMENT '资源类型: knowledgebase/directory/knowledge/member/tag',
                                             resource_uuid VARCHAR(100) COMMENT '资源uuid',
                                             detail JSON COMMENT '操作详情: 包含操作前后的数据、变更字段等',
                                             created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                                             INDEX idx_kb_uuid(knowledgebase_uuid),
                                             INDEX idx_operator(operator),
                                             INDEX idx_operation(operation),
                                             INDEX idx_resource(resource_type, resource_uuid),
                                             INDEX idx_created(created_at),
                                             FOREIGN KEY (knowledgebase_uuid) REFERENCES knowledgebase(uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='知识本操作记录表(审计日志)';
