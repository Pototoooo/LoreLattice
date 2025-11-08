package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/mcp"
	"github.com/Tencent/WeKnora/internal/types"
)

// MCPTool wraps an MCP service tool to implement the Tool interface
type MCPTool struct {
	service    *types.MCPService
	mcpTool    *types.MCPTool
	mcpManager *mcp.MCPManager
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(service *types.MCPService, mcpTool *types.MCPTool, mcpManager *mcp.MCPManager) *MCPTool {
	return &MCPTool{
		service:    service,
		mcpTool:    mcpTool,
		mcpManager: mcpManager,
	}
}

// Name returns the unique name for this tool
// Format: mcp_{service_name}_{tool_name}
func (t *MCPTool) Name() string {
	// Sanitize service name and tool name to create a valid identifier
	serviceName := sanitizeName(t.service.Name)
	toolName := sanitizeName(t.mcpTool.Name)
	return fmt.Sprintf("mcp_%s_%s", serviceName, toolName)
}

// Description returns the tool description
func (t *MCPTool) Description() string {
	serviceDesc := fmt.Sprintf("[MCP Service: %s] ", t.service.Name)
	if t.mcpTool.Description != "" {
		return serviceDesc + t.mcpTool.Description
	}
	return serviceDesc + t.mcpTool.Name
}

// Parameters returns the JSON Schema for tool parameters
func (t *MCPTool) Parameters() map[string]interface{} {
	if t.mcpTool.InputSchema != nil {
		return t.mcpTool.InputSchema
	}

	// Return a default schema if none provided
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute executes the MCP tool
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	logger.GetLogger(ctx).Infof("Executing MCP tool: %s from service: %s", t.mcpTool.Name, t.service.Name)

	// Get or create MCP client
	client, err := t.mcpManager.GetOrCreateClient(t.service)
	if err != nil {
		logger.GetLogger(ctx).Errorf("Failed to get MCP client: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to MCP service: %v", err),
		}, nil
	}

	// Call the tool via MCP
	result, err := client.CallTool(ctx, t.mcpTool.Name, args)
	if err != nil {
		logger.GetLogger(ctx).Errorf("MCP tool call failed: %v", err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Tool execution failed: %v", err),
		}, nil
	}

	// Check if result indicates error
	if result.IsError {
		errorMsg := extractContentText(result.Content)
		logger.GetLogger(ctx).Warnf("MCP tool returned error: %s", errorMsg)
		return &types.ToolResult{
			Success: false,
			Error:   errorMsg,
		}, nil
	}

	// Extract text content from result
	output := extractContentText(result.Content)

	// Build structured data from result
	data := make(map[string]interface{})
	data["content_items"] = result.Content

	logger.GetLogger(ctx).Infof("MCP tool executed successfully: %s", t.mcpTool.Name)

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data:    data,
	}, nil
}

// extractContentText extracts text content from MCP content items
func extractContentText(content []mcp.ContentItem) string {
	var textParts []string

	for _, item := range content {
		switch item.Type {
		case "text":
			if item.Text != "" {
				textParts = append(textParts, item.Text)
			}
		case "image":
			// For images, include a description
			mimeType := item.MimeType
			if mimeType == "" {
				mimeType = "image"
			}
			textParts = append(textParts, fmt.Sprintf("[Image: %s]", mimeType))
		case "resource":
			// For resources, include a reference
			textParts = append(textParts, fmt.Sprintf("[Resource: %s]", item.MimeType))
		default:
			// For other types, try to include any text or data
			if item.Text != "" {
				textParts = append(textParts, item.Text)
			} else if item.Data != "" {
				textParts = append(textParts, fmt.Sprintf("[Data: %s]", item.Type))
			}
		}
	}

	if len(textParts) == 0 {
		return "Tool executed successfully (no text output)"
	}

	return strings.Join(textParts, "\n")
}

// sanitizeName sanitizes a name to create a valid identifier
func sanitizeName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// RegisterMCPTools registers MCP tools from given services
func RegisterMCPTools(registry *ToolRegistry, services []*types.MCPService, mcpManager *mcp.MCPManager) error {
	if len(services) == 0 {
		return nil
	}

	ctx := context.Background()

	for _, service := range services {
		if !service.Enabled {
			continue
		}

		// Get or create client
		client, err := mcpManager.GetOrCreateClient(service)
		if err != nil {
			logger.GetLogger(ctx).Errorf("Failed to create MCP client for service %s: %v", service.Name, err)
			continue
		}

		// List tools from the service
		tools, err := client.ListTools(ctx)
		if err != nil {
			logger.GetLogger(ctx).Errorf("Failed to list tools from MCP service %s: %v", service.Name, err)
			continue
		}

		// Register each tool
		for _, mcpTool := range tools {
			tool := NewMCPTool(service, mcpTool, mcpManager)
			registry.RegisterTool(tool)
			logger.GetLogger(ctx).Infof("Registered MCP tool: %s from service: %s", tool.Name(), service.Name)
		}
	}

	return nil
}

// GetMCPToolsInfo returns information about available MCP tools
func GetMCPToolsInfo(services []*types.MCPService, mcpManager *mcp.MCPManager) (map[string][]string, error) {
	result := make(map[string][]string)
	ctx := context.Background()

	for _, service := range services {
		if !service.Enabled {
			continue
		}

		client, err := mcpManager.GetOrCreateClient(service)
		if err != nil {
			continue
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			continue
		}

		toolNames := make([]string, len(tools))
		for i, tool := range tools {
			toolNames[i] = tool.Name
		}

		result[service.Name] = toolNames
	}

	return result, nil
}

// SerializeMCPToolResult serializes an MCP tool result for display
func SerializeMCPToolResult(result *types.ToolResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("result is nil")
	}

	if !result.Success {
		return fmt.Sprintf("Error: %s", result.Error), nil
	}

	output := result.Output
	if output == "" {
		output = "Success (no output)"
	}

	// If there's structured data, try to format it nicely
	if result.Data != nil {
		if dataBytes, err := json.MarshalIndent(result.Data, "", "  "); err == nil {
			output += "\n\nStructured Data:\n" + string(dataBytes)
		}
	}

	return output, nil
}
