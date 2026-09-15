-- 添加群成员类型 (chat_group) 到知识本成员表
-- Date: 2026-03-23
-- Description: 支持通过如流群添加知识本成员，添加人需要在群中

-- 修改 member_type ENUM 类型，添加 chat_group 选项
-- MySQL 不支持直接修改 ENUM，需要重新定义列
ALTER TABLE knowledgebase_member
MODIFY COLUMN member_type ENUM('username', 'email_group', 'department', 'chat_group') NOT NULL COMMENT '成员类型: username-用户, email_group-邮件组, department-部门, chat_group-群';
