-- 000005_add_web_search_config.up.sql
-- Add web_search_config column to tenants table for web search configuration

BEGIN;

-- Add web_search_config column to tenants table (if not exists)
-- Note: PostgreSQL 9.6+ supports IF NOT EXISTS
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tenants' AND column_name = 'web_search_config'
    ) THEN
        ALTER TABLE tenants 
        ADD COLUMN web_search_config JSONB DEFAULT NULL;
    END IF;
END $$;

-- Add or update comment
COMMENT ON COLUMN tenants.web_search_config IS 'Web search configuration for the tenant';

COMMIT;

