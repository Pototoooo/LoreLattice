-- 增量迁移：添加 agent_config 字段（如果不存在）
-- 创建时间：2025-11-03
-- 目的：为旧版本数据库添加 agent_config 支持
-- 注意：新数据库可能已经有这些字段（通过00-init-db.sql），使用 IF NOT EXISTS 确保兼容

-- ============================================
-- 1. 为 tenants 表添加 agent_config
-- ============================================

-- 检查并添加字段
SET @dbname = DATABASE();
SET @tablename = 'tenants';
SET @columnname = 'agent_config';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      TABLE_SCHEMA = @dbname
      AND TABLE_NAME = @tablename
      AND COLUMN_NAME = @columnname
  ) > 0,
  'SELECT 1', -- 字段已存在，不做任何操作
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname, ' JSON DEFAULT NULL COMMENT ''Tenant-level agent configuration in JSON format''')
));

PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- ============================================
-- 2. 为 sessions 表添加 agent_config  
-- ============================================

SET @dbname = DATABASE();
SET @tablename = 'sessions';
SET @columnname = 'agent_config';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      TABLE_SCHEMA = @dbname
      AND TABLE_NAME = @tablename
      AND COLUMN_NAME = @columnname
  ) > 0,
  'SELECT 1', -- 字段已存在，不做任何操作
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname, ' JSON DEFAULT NULL COMMENT ''Session-level agent configuration in JSON format''')
));

PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- ============================================
-- 3. 为 sessions 表添加 context_config
-- ============================================

SET @dbname = DATABASE();
SET @tablename = 'sessions';
SET @columnname = 'context_config';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      TABLE_SCHEMA = @dbname
      AND TABLE_NAME = @tablename
      AND COLUMN_NAME = @columnname
  ) > 0,
  'SELECT 1', -- 字段已存在，不做任何操作
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname, ' JSON DEFAULT NULL COMMENT ''LLM context management configuration (separate from message storage)''')
));

PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- ============================================
-- 4. 为 messages 表添加 agent_steps
-- ============================================

SET @dbname = DATABASE();
SET @tablename = 'messages';
SET @columnname = 'agent_steps';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      TABLE_SCHEMA = @dbname
      AND TABLE_NAME = @tablename
      AND COLUMN_NAME = @columnname
  ) > 0,
  'SELECT 1', -- 字段已存在，不做任何操作
  CONCAT('ALTER TABLE ', @tablename, ' ADD COLUMN ', @columnname, ' JSON DEFAULT NULL COMMENT ''Agent execution steps (reasoning process and tool calls)''')
));

PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- ============================================
-- 验证：显示所有表的结构
-- ============================================

SELECT 'Verifying tenants table...' as status;
SHOW COLUMNS FROM tenants LIKE 'agent_config';

SELECT 'Verifying sessions table...' as status;
SHOW COLUMNS FROM sessions LIKE 'agent_config';
SHOW COLUMNS FROM sessions LIKE 'context_config';

SELECT 'Verifying messages table...' as status;
SHOW COLUMNS FROM messages LIKE 'agent_steps';

SELECT 'Migration completed successfully!' as status;

