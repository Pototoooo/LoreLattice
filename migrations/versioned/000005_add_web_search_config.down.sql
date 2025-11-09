-- 000005_add_web_search_config.down.sql
-- Remove web_search_config column from tenants table

BEGIN;

-- Remove web_search_config column
ALTER TABLE tenants 
DROP COLUMN IF EXISTS web_search_config;

COMMIT;

