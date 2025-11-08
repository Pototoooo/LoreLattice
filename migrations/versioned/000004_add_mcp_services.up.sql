-- 04-add-mcp-services.sql
-- Create MCP services table for managing external MCP (Model Context Protocol) services

BEGIN;

-- Create mcp_services table
CREATE TABLE IF NOT EXISTS mcp_services (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    transport_type VARCHAR(50) NOT NULL, -- Transport type: sse or http-streamable
    url VARCHAR(512) NOT NULL,
    headers JSONB, -- HTTP headers as JSON object
    auth_config JSONB, -- Authentication configuration (API key, token, custom headers)
    advanced_config JSONB, -- Advanced configuration (timeout, retry settings)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_mcp_services_tenant_id ON mcp_services(tenant_id);
CREATE INDEX IF NOT EXISTS idx_mcp_services_enabled ON mcp_services(enabled);
CREATE INDEX IF NOT EXISTS idx_mcp_services_deleted_at ON mcp_services(deleted_at);

-- Add comment to table
COMMENT ON TABLE mcp_services IS 'MCP service configurations';

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_mcp_services_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_mcp_services_updated_at
    BEFORE UPDATE ON mcp_services
    FOR EACH ROW
    EXECUTE FUNCTION update_mcp_services_updated_at();

COMMIT;

