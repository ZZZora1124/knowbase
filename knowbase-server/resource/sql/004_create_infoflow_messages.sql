-- 如流群消息记录表
CREATE TABLE `infoflow_messages` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,

  -- 消息基础信息
  `group_id` BIGINT NOT NULL COMMENT '群ID',
  `group_name` VARCHAR(255) COMMENT '群名称',
  `user_id` VARCHAR(32) NOT NULL COMMENT '发送者用户ID',
  `time` BIGINT NOT NULL COMMENT '消息时间戳(毫秒)',
  `message_id` VARCHAR(50) NOT NULL COMMENT '原始消息ID',

  -- 消息头部信息
  `from_user_id` VARCHAR(32) COMMENT '发送者ID',
  `to_id` BIGINT COMMENT '接收者ID',
  `to_type` VARCHAR(20) COMMENT '接收者类型: GROUP/USER',
  `msg_type` VARCHAR(20) COMMENT '消息类型: MIXED/TEXT等',

  -- 消息内容（单条body元素）
  `content_type` VARCHAR(20) NOT NULL COMMENT '内容类型: TEXT/IMAGE/AT/LINK/command等',
  `content_text` TEXT COMMENT '文本内容',
  `content_json` JSON COMMENT '完整的body元素JSON',

  -- 通用字段
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,

  INDEX `idx_group_id` (`group_id`),
  INDEX `idx_user_id` (`user_id`),
  INDEX `idx_time` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='如流群消息记录表';
