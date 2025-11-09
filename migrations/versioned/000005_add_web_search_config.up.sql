-- 000005_add_web_search_config.up.sql
-- Add web_search_config column to tenants table for web search configuration

BEGIN;

-- Add web_search_config column to tenants table
ALTER TABLE tenants 
ADD COLUMN web_search_config JSONB DEFAULT NULL;

-- Add comment
COMMENT ON COLUMN tenants.web_search_config IS 'Web search configuration for the tenant';

COMMIT;

