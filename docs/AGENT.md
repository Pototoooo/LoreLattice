# WeKnora Agent Mode 文档

## 概述

WeKnora Agent Mode 是基于 ReAct (Reasoning + Acting) 框架实现的智能代理系统，能够通过工具调用和迭代推理来回答复杂问题。

### 核心特性

- **ReAct 框架**: Thought → Action → Observation 循环
- **工具系统**: 7个知识库相关工具
- **可选规划**: 可在执行前生成任务计划
- **事件追踪**: 完整的执行过程可视化
- **灵活配置**: 支持多种参数调整

## 架构设计

### Agent 工作流程

```
用户查询
    ↓
[可选] 生成执行计划
    ↓
┌─────────────────┐
│  ReAct 循环开始  │
├─────────────────┤
│ 1. Think (思考) │ → LLM 分析当前状态
│ 2. Act (行动)   │ → 调用工具获取信息
│ 3. Observe (观察)│ → 处理工具返回结果
│ [可选] Reflect  │ → 反思当前步骤
│ 4. Decide (决策)│ → 继续或结束?
└─────────────────┘
    ↓
生成最终答案
```

### 组件结构

```
internal/agent/
├── engine.go          # Agent 执行引擎
├── prompts.go         # System prompts
└── tools/             # 工具系统
    ├── tool.go
    ├── registry.go
    ├── knowledge_search.go
    ├── multi_kb_search.go
    ├── list_knowledge_bases.go
    ├── get_chunk_detail.go
    ├── get_related_chunks.go
    ├── query_knowledge_graph.go
    └── get_document_info.go
```

## 配置说明

### Agent 配置结构

```go
type AgentConfig struct {
    Enabled           bool     // 是否启用 Agent 模式
    EnablePlanning    bool     // 是否先规划
    MaxIterations     int      // 最大迭代次数 (1-20)
    ReflectionEnabled bool     // 是否启用反思
    AllowedTools      []string // 允许的工具列表
    Temperature       float64  // LLM 温度 (0-2)
    ThinkingModelID   string   // 推理模型 ID
    KnowledgeBases    []string // 可访问的知识库 ID
}
```

### 全局配置 (config.yaml)

```yaml
agent:
  enabled: true
  default_max_iterations: 5
  default_temperature: 0.7
  reflection_enabled: false
  default_tools:
    - knowledge_search
    - multi_kb_search
    - list_knowledge_bases
    - get_chunk_detail
    - get_related_chunks
    - query_knowledge_graph
    - get_document_info
```

## 可用工具

### 1. knowledge_search
搜索指定知识库中的相关内容。

**参数:**
- `knowledge_base_id` (必需): 知识库ID
- `query` (必需): 搜索查询内容
- `top_k` (可选): 返回结果数量，默认5

**示例:**
```
knowledge_search(knowledge_base_id="kb123", query="什么是RAG", top_k=5)
```

### 2. multi_kb_search
在多个知识库中智能搜索，自动选择最相关的知识库。

**参数:**
- `query` (必需): 搜索查询内容
- `top_k` (可选): 每个知识库返回的结果数量，默认3

**适用场景:** 跨知识库查询，不确定信息在哪个知识库中

### 3. list_knowledge_bases
列出当前可访问的所有知识库。

**参数:** 无

**用途:** 了解有哪些知识库可以搜索

### 4. get_chunk_detail
获取指定chunk的详细信息，包括完整内容、来源文档。

**参数:**
- `chunk_id` (必需): Chunk ID

**用途:** 当搜索结果不够详细时获取完整内容

### 5. get_related_chunks
获取与指定chunk相关的其他chunks。

**参数:**
- `chunk_id` (必需): Chunk ID
- `relation_type` (可选): "sequential" (顺序) 或 "semantic" (语义)
- `limit` (可选): 返回数量，默认5

**用途:** 发现相关信息，扩展上下文

### 6. query_knowledge_graph
查询知识图谱中的实体和关系。

**参数:**
- `knowledge_base_id` (必需): 知识库ID
- `query` (必需): 查询内容（实体名称或查询文本）

**前提:** 知识库已配置知识图谱抽取

### 7. get_document_info
获取文档的元数据信息。

**参数:**
- `knowledge_id` (必需): 文档/知识ID

**用途:** 了解文档的整体情况

## 使用指南

### 创建 Agent Session

```bash
POST /api/v1/sessions

{
  "knowledge_base_id": "kb123",
  "session_strategy": {
    "summary_model_id": "model123",
    ...
  },
  "agent_config": {
    "enabled": true,
    "enable_planning": false,
    "max_iterations": 5,
    "reflection_enabled": false,
    "allowed_tools": [
      "knowledge_search",
      "multi_kb_search",
      "list_knowledge_bases"
    ],
    "temperature": 0.7,
    "knowledge_bases": ["kb123", "kb456"]
  }
}
```

### 发起 Agent 查询

```bash
POST /api/v1/sessions/{session_id}/agent-qa

{
  "query": "请解释RAG技术的工作原理"
}
```

### 响应格式

Agent 会返回 SSE (Server-Sent Events) 流式响应：

1. **知识引用** (可选)
```json
{
  "response_type": "references",
  "knowledge_references": [...]
}
```

2. **Agent 思考过程** (可选，用于调试)
```json
{
  "response_type": "agent_thought",
  "content": "我需要先搜索相关知识..."
}
```

3. **工具调用** (可选)
```json
{
  "response_type": "agent_action",
  "content": "调用工具: knowledge_search"
}
```

4. **最终答案**
```json
{
  "response_type": "answer",
  "content": "RAG技术的工作原理是...",
  "done": true
}
```

## 最佳实践

### 1. 工具选择策略

- **已知知识库**: 使用 `knowledge_search`
- **不确定位置**: 使用 `multi_kb_search`
- **需要上下文**: 使用 `get_related_chunks`
- **探索阶段**: 先用 `list_knowledge_bases`

### 2. 迭代次数设置

- **简单查询**: 3-5 次
- **复杂问题**: 5-10 次
- **探索性任务**: 10-15 次
- **最大限制**: 20 次 (防止无限循环)

### 3. 温度参数

- **精确查询**: 0.3-0.5 (更确定性)
- **创造性任务**: 0.7-1.0 (更多样性)
- **默认推荐**: 0.7

### 4. Planning 启用时机

- **复杂多步骤任务**: 启用
- **简单直接查询**: 禁用
- **探索性问题**: 启用

### 5. Reflection 启用时机

- **关键任务**: 启用 (提高准确性)
- **快速响应**: 禁用 (减少延迟)
- **默认**: 禁用

## 工作原理详解

### ReAct Prompt Template

```
你是一个智能知识库助手。你的任务是通过使用提供的工具来回答用户问题。

工作流程：
1. 分析用户问题，确定需要什么信息
2. 使用合适的工具获取信息（可以多次调用不同工具）
3. 基于获取的信息，提供准确、完整的答案

注意事项：
- 优先使用 multi_kb_search 进行跨知识库搜索
- 如果需要特定知识库，先用 list_knowledge_bases 查看可用知识库
- 如果搜索结果不够详细，使用 get_chunk_detail 获取完整内容
- 使用 get_related_chunks 发现相关信息
- 如果涉及实体关系，使用 query_knowledge_graph
- 引用信息时，说明来源（chunk_id 或 knowledge_base）
- 如果找不到相关信息，诚实告知用户

当前可访问的知识库：
{knowledge_bases}

{plan_context}
```

### 工具调用解析

Agent 会尝试从 LLM 输出中解析工具调用，支持的格式：

```
tool_name(arg1="value1", arg2="value2")
```

例如：
```
knowledge_search(knowledge_base_id="kb123", query="RAG技术")
```

### 循环控制

Agent 会在以下情况停止：

1. LLM 输出包含 "最终答案" 或 "Final Answer"
2. 达到最大迭代次数
3. 工具调用失败且无法恢复
4. 检测到重复循环模式

## 事件追踪

Agent 执行过程中会发送以下事件 (通过 Event Bus):

### agent.plan
```go
type AgentPlanData struct {
    Query    string
    Plan     []string
    Duration int64
}
```

### agent.step
```go
type AgentStepData struct {
    Iteration int
    Thought   string
    ToolCalls []ToolCall
    Duration  int64
}
```

### agent.tool
```go
type AgentActionData struct {
    Iteration  int
    ToolName   string
    ToolInput  map[string]interface{}
    ToolOutput string
    Success    bool
    Error      string
    Duration   int64
}
```

## 示例场景

### 场景 1: 简单知识查询

**用户问题**: "什么是 RAG?"

**Agent 执行流程**:
1. **Think**: 需要搜索 RAG 相关知识
2. **Act**: `multi_kb_search(query="RAG")`
3. **Observe**: 获得3条相关结果
4. **Think**: 信息足够，可以回答
5. **Final Answer**: 基于搜索结果生成答案

**迭代次数**: 2

### 场景 2: 跨文档关联查询

**用户问题**: "比较 RAG 和微调的优缺点"

**Agent 执行流程**:
1. **Think**: 需要分别查找 RAG 和微调的信息
2. **Act**: `multi_kb_search(query="RAG优缺点")`
3. **Observe**: 获得 RAG 相关信息
4. **Think**: 还需要微调的信息
5. **Act**: `multi_kb_search(query="模型微调优缺点")`
6. **Observe**: 获得微调相关信息
7. **Think**: 信息完整，进行对比
8. **Final Answer**: 综合两者信息生成对比答案

**迭代次数**: 4

### 场景 3: 深入探索

**用户问题**: "详细解释向量数据库的工作原理"

**Agent 执行流程**:
1. **Think**: 先查看有哪些知识库
2. **Act**: `list_knowledge_bases()`
3. **Observe**: 发现有"数据库技术"知识库
4. **Think**: 在该知识库中搜索
5. **Act**: `knowledge_search(knowledge_base_id="db_kb", query="向量数据库")`
6. **Observe**: 找到相关 chunk
7. **Think**: 需要更多细节
8. **Act**: `get_chunk_detail(chunk_id="chunk123")`
9. **Observe**: 获得完整内容
10. **Think**: 查看相关内容
11. **Act**: `get_related_chunks(chunk_id="chunk123", relation_type="sequential")`
12. **Observe**: 获得上下文
13. **Final Answer**: 基于详细信息生成深入解释

**迭代次数**: 7

## 常见问题

### Q: Agent 和普通 RAG 有什么区别？

**A**: 
- **普通 RAG**: 一次性检索 → 生成答案
- **Agent**: 可以多次迭代，根据中间结果调整策略，调用不同工具

### Q: 什么时候使用 Agent 模式？

**A**: 
- 需要多步推理的复杂问题
- 需要在多个知识库中查找信息
- 需要深入探索和关联分析
- 问题模糊，需要澄清和逐步细化

### Q: Agent 模式的成本如何？

**A**:
- **Token 消耗**: 比普通 RAG 高 (多次 LLM 调用)
- **响应时间**: 较长 (迭代执行)
- **准确性**: 通常更高 (多步验证)

### Q: 如何优化 Agent 性能？

**A**:
1. 合理设置 `max_iterations` (避免过多)
2. 选择必要的工具 (`allowed_tools`)
3. 禁用不需要的功能 (planning, reflection)
4. 使用更快的模型作为 thinking_model

### Q: 工具调用失败怎么办？

**A**: Agent 会：
1. 在 Observation 中记录错误
2. 尝试使用其他工具
3. 如果多次失败，会在最终答案中说明

## 技术限制与注意事项

### 当前限制

1. **工具调用解析**: 使用简单的模式匹配，可能无法处理复杂格式
2. **Function Calling**: 当前使用 prompt-based 方式，未来可升级为原生 function calling
3. **并行工具调用**: 当前不支持，工具按顺序执行
4. **图谱查询**: 当前使用 hybrid search，完整图谱功能开发中

### 未来改进

- [ ] 支持 LLM 原生 function calling (GPT-4, Claude 等)
- [ ] 并行工具执行
- [ ] 更智能的循环检测
- [ ] 工具结果缓存
- [ ] 自定义工具注册
- [ ] Agent 执行可视化 UI

## 开发指南

### 创建自定义工具

```go
package tools

import (
    "context"
    "github.com/Tencent/WeKnora/internal/types"
)

type CustomTool struct {
    BaseTool
    // 添加依赖
}

func NewCustomTool() *CustomTool {
    return &CustomTool{
        BaseTool: NewBaseTool(
            "custom_tool",
            "工具描述",
        ),
    }
}

func (t *CustomTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type":        "string",
                "description": "参数描述",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *CustomTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
    // 实现工具逻辑
    param1 := args["param1"].(string)
    
    // 执行操作
    result := doSomething(param1)
    
    return &types.ToolResult{
        Success: true,
        Output:  result,
        Data:    map[string]interface{}{},
    }, nil
}
```

### 注册自定义工具

在 `agent_service.go` 的 `registerTools` 方法中添加：

```go
case "custom_tool":
    registry.RegisterTool(tools.NewCustomTool())
```

## 总结

WeKnora Agent Mode 提供了强大的迭代推理能力，适用于复杂的知识查询场景。通过合理配置和工具选择，可以在准确性和效率之间找到最佳平衡点。

如有问题或建议，请提交 Issue 或 Pull Request。

