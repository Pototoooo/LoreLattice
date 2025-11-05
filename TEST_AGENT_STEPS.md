# 测试Agent Steps功能

## 前置条件

1. 运行数据库迁移脚本：
```bash
# MySQL
mysql -u user -p database < migrations/mysql/05-add-agent-steps-to-messages.sql

# ParadeDB/PostgreSQL
psql -U user -d database -f migrations/paradedb/05-add-agent-steps-to-messages.sql
```

2. 启动服务器：
```bash
./server
```

## 测试步骤

### 1. 创建启用Agent的会话

```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 1" \
  -d '{
    "knowledge_base_id": "your-kb-id"
  }'
```

### 2. 发送Agent查询（使用Agent模式）

```bash
curl -X POST http://localhost:8080/api/v1/sessions/{session_id}/agent/qa \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 1" \
  -d '{
    "query": "帮我搜索关于机器学习的知识"
  }'
```

### 3. 加载消息历史并检查AgentSteps

```bash
curl http://localhost:8080/api/v1/sessions/{session_id}/messages?limit=10 \
  -H "X-Tenant-ID: 1"
```

### 预期响应格式

```json
{
  "success": true,
  "data": [
    {
      "id": "message-uuid",
      "session_id": "session-uuid",
      "role": "user",
      "content": "帮我搜索关于机器学习的知识",
      "knowledge_references": [],
      "is_completed": true,
      "created_at": "2025-10-31T10:00:00Z",
      "updated_at": "2025-10-31T10:00:00Z"
    },
    {
      "id": "message-uuid-2",
      "session_id": "session-uuid",
      "role": "assistant",
      "content": "根据搜索结果，机器学习是...",
      "knowledge_references": [
        {
          "id": "chunk-id",
          "content": "...",
          "score": 0.95,
          "knowledge_id": "doc-id",
          "knowledge_title": "机器学习基础"
        }
      ],
      "agent_steps": [
        {
          "iteration": 0,
          "thought": "用户想了解机器学习，我需要搜索相关知识库",
          "tool_calls": [
            {
              "id": "call_123",
              "name": "search_knowledge",
              "args": {
                "query": "机器学习",
                "knowledge_bases": ["kb-id"]
              },
              "result": {
                "success": true,
                "output": "找到5个相关文档",
                "data": {
                  "knowledge_refs": [...]
                }
              },
              "duration": 150
            }
          ],
          "observations": [
            "找到5个相关文档"
          ],
          "timestamp": "2025-10-31T10:00:01Z"
        }
      ],
      "is_completed": true,
      "created_at": "2025-10-31T10:00:00Z",
      "updated_at": "2025-10-31T10:00:02Z"
    }
  ]
}
```

## 验证要点

### ✅ 用户对话历史（Message表）
- assistant消息包含`agent_steps`字段
- 每个step包含：
  - `iteration`: 迭代次数
  - `thought`: Agent的思考内容
  - `tool_calls`: 调用的工具列表
  - `observations`: 观察结果
  - `timestamp`: 时间戳
- user消息不包含`agent_steps`字段（使用omitempty）

### ✅ 大模型上下文分离
后续对话时，传给LLM的上下文应该是：
```go
// 只包含简化的Role+Content
[
  {"role": "user", "content": "帮我搜索关于机器学习的知识"},
  {"role": "assistant", "content": "根据搜索结果，机器学习是..."}
  // 不包含agent_steps信息
]
```

可以通过日志验证：
```
INFO: Loaded 2 history messages for agent context
```

### ✅ 数据库验证

```sql
-- 检查agent_steps列
SELECT id, role, 
       CASE 
         WHEN agent_steps IS NULL THEN 'NULL'
         ELSE JSON_EXTRACT(agent_steps, '$[0].iteration')
       END as first_step_iteration
FROM messages 
WHERE session_id = 'your-session-id'
ORDER BY created_at DESC;
```

## 常见问题

### Q1: agent_steps字段为空？
检查：
- Agent模式是否启用（使用`/agent/qa`端点）
- 数据库迁移是否执行成功
- Handler的事件监听器是否注册

### Q2: 上下文是否包含agent_steps？
- 不应该包含！
- 检查`convertMessagesToChatHistory`函数
- 确保只使用`Role`和`Content`字段

### Q3: 性能影响？
- `agent_steps`使用JSON存储，查询效率高
- 前端可以按需展示或折叠步骤详情
- 添加了`idx_messages_session_role`索引优化查询

## 前端集成建议

```typescript
interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  agent_steps?: AgentStep[];  // 可选字段
  knowledge_references?: Reference[];
}

interface AgentStep {
  iteration: number;
  thought: string;
  tool_calls: ToolCall[];
  observations: string[];
  timestamp: string;
}

// 展示Agent思考过程
function renderAgentSteps(steps: AgentStep[]) {
  return steps.map(step => (
    <div key={step.iteration}>
      <h4>步骤 {step.iteration + 1}</h4>
      <p>思考: {step.thought}</p>
      <ul>
        {step.tool_calls.map(call => (
          <li key={call.id}>
            调用: {call.name} - {call.result.output}
          </li>
        ))}
      </ul>
    </div>
  ));
}
```

## 总结

这个实现成功地：
1. ✅ 存储了Agent的详细执行步骤
2. ✅ 每次任务只记录一个Q和A
3. ✅ 分离了用户历史和LLM上下文
4. ✅ 避免了上下文重复
5. ✅ 保持了向后兼容性

