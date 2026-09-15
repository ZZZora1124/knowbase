-- 添加如流群消息删除审计日志表
-- Date: 2026-04-01
-- Description: 记录群聊消息删除操作的审计信息，支持手动删除和定时清理两种场景

CREATE TABLE `infoflow_message_deletion_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` bigint(20) NOT NULL DEFAULT 0 COMMENT '群ID（0表示定时清理全局删除）',
  `day_before` int(11) NOT NULL DEFAULT 0 COMMENT '删除N天之前的数据（0表示删除该群所有数据）',
  `deleted_count` bigint(20) NOT NULL DEFAULT 0 COMMENT '删除的消息记录数',
  `operator` varchar(64) NOT NULL DEFAULT '' COMMENT '操作人（定时任务为system）',
  `operation_type` varchar(20) NOT NULL DEFAULT 'manual' COMMENT '操作类型：manual-手动删除, scheduled-定时清理',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_operation_type` (`operation_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='如流群消息删除审计日志表';
