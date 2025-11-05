package tools

import (
	"context"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// ThinkingTool implements the thinking tool functionality
// This is a no-op tool inspired by code-agent and tau-bench
// It logs thoughts for transparency and better decision-making
type ThinkingTool struct {
	BaseTool
}

// NewThinkingTool creates a new thinking tool instance
func NewThinkingTool() *ThinkingTool {
	description := `Deep reasoning tool for systematic thinking. Use this frequently to analyze, plan, and reflect.

## Purpose

This tool helps you:
- **Analyze problems** before acting
- **Plan your approach** for complex tasks
- **Evaluate results** after tool calls
- **Reflect** on strategy and adjust

This is a no-op tool - it doesn't fetch data, but helps organize your reasoning transparently.

## When to Use

**Use frequently at key moments**:
- **Start**: Analyze the question, understand what's needed
- **Before major decisions**: Which tool to use? What query to run?
- **After tool calls**: Evaluate results, decide next step
- **Before answering**: Organize information, check completeness

## What to Think About

### At Start
- What is the user really asking?
- What information do I need?
- What's my search strategy?

### After Search Results
- Are results relevant? Complete?
- Do I need more context (get_related_chunks)?
- Should I search more or answer now?

### Before Answering
- Do I have enough information?
- Are all sources cited?
- Is the structure clear?


## Tips

- Use before important decisions
- Keep thoughts focused and actionable
- End with clear next step
- Don't overthink simple queries`

	return &ThinkingTool{
		BaseTool: NewBaseTool(
			"thinking",
			description,
		),
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *ThinkingTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"thought": map[string]interface{}{
				"type":        "string",
				"description": "Your thinking process and reasoning content",
			},
		},
		"required": []string{"thought"},
	}
}

// Execute executes the thinking tool
func (t *ThinkingTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	logger.Infof(ctx, "[Tool][Thinking] Execute started")

	thought, ok := args["thought"].(string)
	if !ok {
		thought = "Thought content not provided"
		logger.Warnf(ctx, "[Tool][Thinking] No thought content provided")
	}

	logger.Infof(ctx, "[Tool][Thinking] Thought length: %d characters", len(thought))
	logger.Debugf(ctx, "[Tool][Thinking] Thought content:\n%s", thought)

	// This is a no-op tool - it just logs the thought
	// The thought itself is valuable for transparency and decision-making

	// Format output for better display
	output := "Thought process recorded:\n\n"
	output += thought

	logger.Infof(ctx, "[Tool][Thinking] Execute completed")
	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"thought":      thought,
			"display_type": "thinking",
		},
	}, nil
}
