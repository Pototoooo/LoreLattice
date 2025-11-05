package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

// TodoWriteTool implements a planning tool for complex tasks
// This is an optional tool that helps organize multi-step research
type TodoWriteTool struct {
	BaseTool
}

// PlanStep represents a single step in the research plan
type PlanStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	ToolsToUse  string `json:"tools_to_use"`
	Status      string `json:"status"` // pending, in_progress, completed, skipped
}

// NewTodoWriteTool creates a new todo_write tool instance
func NewTodoWriteTool() *TodoWriteTool {
	description := `Use this tool to create a structured, actionable plan for complex research tasks. This helps you organize multi-step investigations, track progress, and ensure comprehensive coverage.

## Critical Thinking Integration
**IMPORTANT**: When working with complex research questions, you should typically use the **think tool** for deep analysis BEFORE creating your plan. This ensures:

- **Problem Analysis**: Use think tool to analyze the research question, identify key dimensions, and explore different angles
- **Strategic Planning**: Use think tool to consider what information is needed, which knowledge bases to search, and potential challenges
- **Scope Definition**: Use think tool to determine the depth and breadth of research required

The think tool helps ensure your plan is comprehensive, well-structured, and addresses all aspects of the research question effectively.

## When to Use This Tool

Use this tool proactively in these scenarios:

1. **Complex multi-dimensional questions** - Questions requiring investigation of multiple aspects or dimensions
2. **Comparison tasks** - When you need to compare multiple systems, approaches, or solutions systematically
3. **Deep research tasks** - Questions requiring exploration of multiple knowledge areas or documents
4. **Architecture analysis** - Tasks involving system design, architecture review, or technical deep-dives
5. **User requests explicit planning** - When users ask for structured analysis or step-by-step investigation

## When NOT to Use This Tool

Skip using this tool when:

1. **Simple, direct queries** - Questions with straightforward answers (e.g., "What is a vector database?")
2. **Single-step searches** - When you already know exactly what to search for
3. **Mid-execution** - Don't create plans after you've already started searching
4. **Trivial lookups** - Simple fact-finding that requires one tool call

NOTE: If the question can be answered with 1-2 simple searches, skip planning and search directly.

## Examples of When to Use

<example>
User: "如何设计一个高可用的微服务架构？需要考虑哪些方面？"
Assistant: This is a complex, multi-dimensional question. Let me first think through the key aspects and then create a structured research plan.
*Uses think tool to analyze: architecture patterns, reliability concerns, deployment strategies, monitoring needs*
Assistant: Based on my analysis, let me create a research plan covering the main dimensions.
*Creates plan with steps for:*
1. Search microservices architecture patterns and best practices
2. Investigate high availability strategies (load balancing, failover, redundancy)
3. Research service mesh and communication patterns
4. Explore monitoring and observability solutions
5. Investigate deployment and scaling strategies
*Begins executing the plan*

<reasoning>
The assistant used both think and todo_write tools because:
1. First used think tool to analyze the multi-dimensional nature of the question
2. Identified key areas: architecture, availability, communication, monitoring, deployment
3. Created a structured plan to ensure systematic coverage
4. The plan helps avoid missing critical aspects of high availability
</reasoning>
</example>

<example>
User: "对比Redis、Memcached和Hazelcast这三种缓存系统的性能、成本和适用场景"
Assistant: This is a structured comparison task. Let me think through the comparison dimensions and create a systematic research plan.
*Uses think tool to identify: performance metrics, cost factors, use case categories, comparison criteria*
Assistant: I'll create a research plan to ensure fair, comprehensive comparison across all three systems.
*Creates plan with steps:*
1. Search Redis characteristics (performance, features, use cases)
2. Search Memcached characteristics (performance, features, use cases)
3. Search Hazelcast characteristics (performance, features, use cases)
4. Search direct performance comparisons between these systems
5. Investigate cost considerations and deployment complexity
*Executes searches systematically*

<reasoning>
The assistant used todo_write because:
1. Comparison tasks benefit from structured, parallel research
2. The plan ensures each system is investigated equally
3. Structured approach prevents bias toward any particular system
4. Makes it easier to synthesize structured comparison at the end
</reasoning>
</example>

## Examples of When NOT to Use

<example>
User: "向量数据库的主要用途是什么？"
Assistant: Let me search for information about vector database use cases.
*Uses knowledge_search directly without planning*

<reasoning>
The assistant skipped todo_write because:
1. This is a simple, direct question
2. One well-crafted search will likely provide the answer
3. No need to organize multiple research steps
4. Planning would add unnecessary overhead
</reasoning>
</example>

<example>
User: "Kubernetes是什么？"
Assistant: Let me search for information about Kubernetes.
*Uses knowledge_search directly*

<reasoning>
Simple definitional question - just search directly without planning.
</reasoning>
</example>

## Standard Workflow

The recommended workflow when using this tool:

` + "```" + `
Step 1: User asks complex question
   ↓
Step 2: Use think tool (analyze problem, identify dimensions, consider approaches)
   ↓
Step 3: Use todo_write tool (create structured plan with steps, tools, priorities) [OPTIONAL]
   ↓
Step 4: Execute plan steps (use search tools, mark progress)
   ↓
Step 5: Use think tool (evaluate results, identify gaps)
   ↓
Step 6: Adjust plan or execute remaining steps
   ↓
Step 7: Synthesize comprehensive answer with citations
` + "```" + `

## Plan Structure

Your plan should include steps with:
- **ID**: Unique identifier for each step (e.g., "step1", "step2")
- **Description**: Clear description of what to investigate
- **Tools to Use**: Which tools to use (e.g., "knowledge_search", "get_related_chunks")
- **Status**: Current status (pending, in_progress, completed, skipped)

## Step Status Management

1. **Status Values**:
   - **pending**: Step not yet started (default for new steps)
   - **in_progress**: Currently executing this step
   - **completed**: Step finished successfully with results
   - **skipped**: Step determined to be unnecessary based on findings

2. **Status Management Best Practices**:
   - Start with all steps as "pending"
   - Mark one step as "in_progress" before executing it
   - Update to "completed" immediately after finishing
   - Mark as "skipped" if findings make a step unnecessary

## Parameters

- **task** (required): The complex task or question you need to create a plan for
  - Be specific about what needs to be accomplished
  - Include the user's actual question or research goal

- **steps** (required): Array of plan steps, each containing:
  - **id**: Unique identifier (e.g., "step1", "step2")
  - **description**: What to investigate or accomplish
  - **tools_to_use**: Suggested tools for this step
  - **status**: Current status (pending, in_progress, completed, skipped)

## Best Practices

1. **Think First**: Always use think tool before creating a plan to ensure thorough analysis
2. **Be Specific**: Each step should have a clear, actionable objective
3. **Stay Flexible**: Adjust the plan based on findings - some steps may become unnecessary
4. **Track Progress**: Update step status as you work through the plan
5. **Don't Over-plan**: 3-7 steps is usually sufficient; more suggests over-complexity
`

	return &TodoWriteTool{
		BaseTool: NewBaseTool("todo_write", description),
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *TodoWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The complex task or question you need to create a plan for",
			},
			"steps": map[string]interface{}{
				"type":        "array",
				"description": "Array of research plan steps with status tracking",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique identifier for this step (e.g., 'step1', 'step2')",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Clear description of what to investigate or accomplish in this step",
						},
						"tools_to_use": map[string]interface{}{
							"type":        "string",
							"description": "Suggested tools for this step (e.g., 'knowledge_search', 'get_related_chunks')",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed", "skipped"},
							"description": "Current status: pending (not started), in_progress (executing), completed (finished), skipped (unnecessary)",
						},
					},
					"required": []string{"id", "description", "status"},
				},
			},
		},
		"required": []string{"task", "steps"},
	}
}

// Execute executes the todo_write tool
func (t *TodoWriteTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	task, ok := args["task"].(string)
	if !ok {
		task = "未提供任务描述"
	}

	// Parse plan steps
	var planSteps []PlanStep
	if stepsData, ok := args["steps"].([]interface{}); ok {
		for _, stepData := range stepsData {
			if stepMap, ok := stepData.(map[string]interface{}); ok {
				step := PlanStep{
					ID:          getStringField(stepMap, "id"),
					Description: getStringField(stepMap, "description"),
					ToolsToUse:  getStringField(stepMap, "tools_to_use"),
					Status:      getStringField(stepMap, "status"),
				}
				planSteps = append(planSteps, step)
			}
		}
	}

	// Generate formatted output
	output := generatePlanOutput(task, planSteps)

	// Prepare structured data for response
	stepsJSON, _ := json.Marshal(planSteps)

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"task":         task,
			"steps":        planSteps,
			"steps_json":   string(stepsJSON),
			"total_steps":  len(planSteps),
			"plan_created": true,
			"display_type": "plan",
		},
	}, nil
}

// Helper function to safely get string field from map
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// generatePlanOutput generates a formatted plan output
func generatePlanOutput(task string, steps []PlanStep) string {
	output := "计划已创建\n\n"
	output += fmt.Sprintf("**任务**: %s\n\n", task)

	if len(steps) == 0 {
		output += "注意：未提供具体步骤。建议创建3-7个结构化步骤以系统化研究。\n\n"
		output += "建议的通用流程：\n"
		output += "1. 使用 knowledge_search 进行初步信息收集\n"
		output += "2. 使用 get_related_chunks 获取关键信息详情\n"
		output += "3. 使用 get_related_chunks 扩展上下文理解\n"
		output += "4. 使用 think 工具评估结果并综合答案\n"
		return output
	}

	output += "**计划步骤**:\n\n"

	// Display all steps in order
	for i, step := range steps {
		output += formatPlanStep(i+1, step)
	}

	output += "\n**执行指南**:\n"
	output += "- 每步执行前标记为 in_progress，完成后标记为 completed\n"
	output += "- 根据搜索结果灵活调整计划，可跳过不必要的步骤\n"
	output += "- 在关键决策点使用 think 工具深入分析\n"
	output += "- 如果某一步骤已获得足够信息，可跳过后续步骤\n\n"
	output += "注意：计划是指导而非硬性要求，保持灵活应对。"

	return output
}

// formatPlanStep formats a single plan step for output
func formatPlanStep(index int, step PlanStep) string {
	statusEmoji := map[string]string{
		"pending":     "⏳",
		"in_progress": "🔄",
		"completed":   "✅",
		"skipped":     "⏭️",
	}

	emoji, ok := statusEmoji[step.Status]
	if !ok {
		emoji = "⏳"
	}

	output := fmt.Sprintf("  %d. %s [%s] %s\n", index, emoji, step.Status, step.Description)

	if step.ToolsToUse != "" {
		output += fmt.Sprintf("     工具: %s\n", step.ToolsToUse)
	}

	return output
}
