-- Add web_search_config column to tenants table
-- This migration adds support for web search configuration at tenant level

ALTER TABLE tenants 
ADD COLUMN web_search_config JSONB DEFAULT NULL;

