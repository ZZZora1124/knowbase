-- 为 knowledgebase_member 表添加 member_type 和 member_value 的复合索引
-- 用于优化通过成员类型和成员值批量查询知识本UUID的性能
-- 适用场景: ListMyKnowledgebases 接口查询邮件组和部门成员关系

ALTER TABLE knowledgebase_member
ADD INDEX idx_member_type_value(member_type, member_value);

-- 索引说明:
-- 1. 优化查询: SELECT DISTINCT knowledgebase_uuid FROM knowledgebase_member
--              WHERE member_type = ? AND member_value IN (?)
-- 2. 覆盖索引: 可以仅通过索引获取 knowledgebase_uuid，无需回表
-- 3. 与现有索引的关系:
--    - uk_kb_member(knowledgebase_uuid, member_type, member_value): 保证唯一性
--    - idx_member_type_value(member_type, member_value): 优化反向查询（从成员找知识本）
