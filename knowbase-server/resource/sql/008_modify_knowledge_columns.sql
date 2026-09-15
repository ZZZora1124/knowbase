-- 修改 knowledge 表的字段长度
-- 原因：知识类型和知识源的枚举值可能扩展，增加字段长度以适应未来需求
-- 新增知识类型：readme（与 content 类型使用相同逻辑）

-- 修改知识类型字段长度：从 ENUM 改为 VARCHAR(36)
-- 修改知识源字段长度：从 ENUM 改为 VARCHAR(36)
ALTER TABLE knowledge
MODIFY COLUMN knowledge_type VARCHAR(36) NOT NULL COMMENT '知识类型: content, readme, text_file, audio_file, image_file, video_file, config',
MODIFY COLUMN knowledge_source VARCHAR(36) DEFAULT 'custom' COMMENT '知识来源: custom, icode_repo, icode_deepwiki, icafe_card, ku_doc, github_repo, infoflow_group, iapi_doc, knowledge_book';
