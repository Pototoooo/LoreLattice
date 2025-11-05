-- 增量迁移：添加 agent_config 字段（如果不存在）
-- 创建时间：2025-11-03
-- 目的：为旧版本数据库添加 agent_config 支持
-- 注意：新数据库可能已经有这些字段（通过01-migrate-to-paradedb.sql），使用 IF NOT EXISTS 确保兼容

-- ============================================
-- 1. 为 tenants 表添加 agent_config
-- ============================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'tenants' 
        AND column_name = 'agent_config'
    ) THEN
        ALTER TABLE tenants 
        ADD COLUMN agent_config JSONB DEFAULT NULL;
        
        COMMENT ON COLUMN tenants.agent_config IS 'Tenant-level agent configuration in JSON format';
        
        RAISE NOTICE 'Added agent_config column to tenants table';
    ELSE
        RAISE NOTICE 'agent_config column already exists in tenants table';
    END IF;
END $$;

-- ============================================
-- 2. 为 sessions 表添加 agent_config
-- ============================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'sessions' 
        AND column_name = 'agent_config'
    ) THEN
        ALTER TABLE sessions 
        ADD COLUMN agent_config JSONB DEFAULT NULL;
        
        COMMENT ON COLUMN sessions.agent_config IS 'Session-level agent configuration in JSON format';
        
        RAISE NOTICE 'Added agent_config column to sessions table';
    ELSE
        RAISE NOTICE 'agent_config column already exists in sessions table';
    END IF;
END $$;

-- ============================================
-- 3. 为 sessions 表添加 context_config
-- ============================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'sessions' 
        AND column_name = 'context_config'
    ) THEN
        ALTER TABLE sessions 
        ADD COLUMN context_config JSONB DEFAULT NULL;
        
        COMMENT ON COLUMN sessions.context_config IS 'LLM context management configuration (separate from message storage)';
        
        RAISE NOTICE 'Added context_config column to sessions table';
    ELSE
        RAISE NOTICE 'context_config column already exists in sessions table';
    END IF;
END $$;

-- ============================================
-- 4. 为 messages 表添加 agent_steps
-- ============================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'messages' 
        AND column_name = 'agent_steps'
    ) THEN
        ALTER TABLE messages 
        ADD COLUMN agent_steps JSONB DEFAULT NULL;
        
        COMMENT ON COLUMN messages.agent_steps IS 'Agent execution steps (reasoning process and tool calls)';
        
        RAISE NOTICE 'Added agent_steps column to messages table';
    ELSE
        RAISE NOTICE 'agent_steps column already exists in messages table';
    END IF;
END $$;

-- ============================================
-- 5. 为 JSON 字段添加 GIN 索引（提高查询性能）
-- ============================================

DO $$
BEGIN
    -- 为 tenants.agent_config 添加索引
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'tenants' 
        AND indexname = 'idx_tenants_agent_config'
    ) THEN
        CREATE INDEX idx_tenants_agent_config ON tenants USING GIN (agent_config);
        RAISE NOTICE 'Created index idx_tenants_agent_config';
    ELSE
        RAISE NOTICE 'Index idx_tenants_agent_config already exists';
    END IF;
    
    -- 为 sessions.agent_config 添加索引
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'sessions' 
        AND indexname = 'idx_sessions_agent_config'
    ) THEN
        CREATE INDEX idx_sessions_agent_config ON sessions USING GIN (agent_config);
        RAISE NOTICE 'Created index idx_sessions_agent_config';
    ELSE
        RAISE NOTICE 'Index idx_sessions_agent_config already exists';
    END IF;
    
    -- 为 sessions.context_config 添加索引
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'sessions' 
        AND indexname = 'idx_sessions_context_config'
    ) THEN
        CREATE INDEX idx_sessions_context_config ON sessions USING GIN (context_config);
        RAISE NOTICE 'Created index idx_sessions_context_config';
    ELSE
        RAISE NOTICE 'Index idx_sessions_context_config already exists';
    END IF;
    
    -- 为 messages.agent_steps 添加索引
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'messages' 
        AND indexname = 'idx_messages_agent_steps'
    ) THEN
        CREATE INDEX idx_messages_agent_steps ON messages USING GIN (agent_steps);
        RAISE NOTICE 'Created index idx_messages_agent_steps';
    ELSE
        RAISE NOTICE 'Index idx_messages_agent_steps already exists';
    END IF;
END $$;

-- ============================================
-- 验证：显示所有表的字段
-- ============================================

SELECT 'Verifying tenants table...' as status;
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'tenants' AND column_name = 'agent_config';

SELECT 'Verifying sessions table...' as status;
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'sessions' AND column_name IN ('agent_config', 'context_config');

SELECT 'Verifying messages table...' as status;
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'messages' AND column_name = 'agent_steps';

SELECT 'Migration completed successfully!' as status;

