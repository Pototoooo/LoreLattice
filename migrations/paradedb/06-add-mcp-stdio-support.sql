-- 06-add-mcp-stdio-support.sql
-- Add stdio transport support for MCP services

-- Add stdio_config and env_vars columns
ALTER TABLE mcp_services 
ADD COLUMN IF NOT EXISTS stdio_config JSONB, -- Stdio configuration: {command: "uvx"/"npx", args: [...]}
ADD COLUMN IF NOT EXISTS env_vars JSONB;     -- Environment variables for stdio transport

-- Make url column optional (remove NOT NULL constraint)
ALTER TABLE mcp_services 
ALTER COLUMN url DROP NOT NULL;

-- Add check constraint: stdio transport requires stdio_config, others require url
ALTER TABLE mcp_services
ADD CONSTRAINT chk_mcp_transport_config CHECK (
    (transport_type = 'stdio' AND stdio_config IS NOT NULL) OR
    (transport_type != 'stdio' AND url IS NOT NULL)
);

