-- Rollback: Remove MCP services table

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_mcp_services_updated_at ON mcp_services;

-- Drop function
DROP FUNCTION IF EXISTS update_mcp_services_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_mcp_services_tenant_id;
DROP INDEX IF EXISTS idx_mcp_services_enabled;
DROP INDEX IF EXISTS idx_mcp_services_deleted_at;

-- Drop table
DROP TABLE IF EXISTS mcp_services;

COMMIT;

