package tools

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

// ToolRegistry manages the registration and retrieval of tools
type ToolRegistry struct {
	tools                map[string]types.Tool
	knowledgeBaseService interfaces.KnowledgeBaseService
	knowledgeService     interfaces.KnowledgeService
	chunkService         interfaces.ChunkService
	db                   *gorm.DB // gorm.DB interface for database query tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(
	knowledgeBaseService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	chunkService interfaces.ChunkService,
	db *gorm.DB, // gorm.DB for database operations
) *ToolRegistry {
	return &ToolRegistry{
		tools:                make(map[string]types.Tool),
		knowledgeBaseService: knowledgeBaseService,
		knowledgeService:     knowledgeService,
		chunkService:         chunkService,
		db:                   db,
	}
}

// RegisterTool adds a tool to the registry
func (r *ToolRegistry) RegisterTool(tool types.Tool) {
	r.tools[tool.Name()] = tool
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (types.Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// ListTools returns all registered tool names
func (r *ToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetFunctionDefinitions returns function definitions for all registered tools
func (r *ToolRegistry) GetFunctionDefinitions() []types.FunctionDefinition {
	definitions := make([]types.FunctionDefinition, 0)
	for _, tool := range r.tools {
		definitions = append(definitions, types.FunctionDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return definitions
}

// ExecuteTool executes a tool by name with the given arguments
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
	tool, err := r.GetTool(name)
	if err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return tool.Execute(ctx, args)
}
