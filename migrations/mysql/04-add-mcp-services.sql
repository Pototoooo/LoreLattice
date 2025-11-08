-- 04-add-mcp-services.sql
-- Create MCP services table for managing external MCP (Model Context Protocol) services

-- Create mcp_services table
CREATE TABLE IF NOT EXISTS mcp_services (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    transport_type VARCHAR(50) NOT NULL COMMENT 'Transport type: sse or http-streamable',
    url VARCHAR(512) NOT NULL,
    headers JSON COMMENT 'HTTP headers as JSON object',
    auth_config JSON COMMENT 'Authentication configuration (API key, token, custom headers)',
    advanced_config JSON COMMENT 'Advanced configuration (timeout, retry settings)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_enabled (enabled),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='MCP service configurations';

