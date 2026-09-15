-- 目录表新增 is_default 字段，标记知识本的默认目录
-- 每个知识本有且仅有一个 is_default=1 的目录，创建知识本时自动生成
-- is_default=1 的目录不允许被删除
-- directory_uuid=NULL 的历史知识由代码层兼容，无需回填数据库

-- Step 1: 新增 is_default 字段
ALTER TABLE knowledgebase_directory
  ADD COLUMN is_default TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否为默认目录，1=是，0=否';

-- Step 2: 为没有默认目录的历史知识本各插入一条默认目录
-- 直接复用 kb.uuid 作为默认目录的 uuid：一个知识本只有一条默认目录，天然唯一
INSERT INTO knowledgebase_directory (uuid, knowledgebase_uuid, parent_uuid, name, is_default, created_at, updated_at)
SELECT
  kb.uuid      AS uuid,
  kb.uuid      AS knowledgebase_uuid,
  NULL         AS parent_uuid,
  '默认目录'   AS name,
  1            AS is_default,
  NOW()        AS created_at,
  NOW()        AS updated_at
FROM knowledgebase kb
WHERE kb.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM knowledgebase_directory d
    WHERE d.knowledgebase_uuid = kb.uuid
      AND d.is_default = 1
      AND d.deleted_at IS NULL
  );
