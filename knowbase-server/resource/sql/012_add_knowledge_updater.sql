-- 知识表新增更新人字段
-- Date: 2026-04-14
-- Description: 记录最后更新知识的操作者用户名，与 creator_username 对称

ALTER TABLE knowledge ADD updater_username varchar(100) DEFAULT '' NOT NULL AFTER creator_username,
  ADD INDEX idx_updater(updater_username);
