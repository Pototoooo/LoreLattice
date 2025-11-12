-- 06-add-mcp-stdio-support.sql
-- Rollback stdio transport support for MCP services

BEGIN;

-- Remove check constraint
ALTER TABLE mcp_services
DROP CONSTRAINT IF EXISTS chk_mcp_transport_config;

-- Make url column required again
ALTER TABLE mcp_services 
ALTER COLUMN url SET NOT NULL;

-- Remove stdio_config and env_vars columns
ALTER TABLE mcp_services 
DROP COLUMN IF EXISTS env_vars,
DROP COLUMN IF EXISTS stdio_config;

COMMIT;

