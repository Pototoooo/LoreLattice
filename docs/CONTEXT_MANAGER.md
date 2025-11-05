# LLM Context Manager - 大模型上下文管理器

## 概述

LLM Context Manager 是一个专门用于管理大模型对话上下文的组件，它**独立于消息存储系统**，专注于管理发送给大模型的上下文窗口。

### 关键特性

1. **独立管理**: 与消息的数据库存储分离，专门管理发送给 LLM 的上下文
2. **Token 限制管理**: 自动监控和管理上下文的 Token 数量
3. **智能压缩**: 当上下文超出限制时，自动应用压缩策略
4. **按 Session 维护**: 每个会话独立管理其上下文
5. **灵活配置**: 支持会话级别的自定义配置

## 为什么需要 Context Manager？

### 问题背景

- **消息存储** vs **LLM 上下文**: 
  - 消息存储：完整保存所有对话历史，用于展示和审计
  - LLM 上下文：有 Token 限制，需要精简管理
  
- **Token 限制**: 不同模型有不同的上下文窗口限制（如 4K, 8K, 16K tokens）

- **性能优化**: 过长的上下文会增加推理时间和成本

### 解决方案

Context Manager 提供：
- 自动管理上下文窗口大小
- 智能压缩历史消息
- 保留最重要的上下文信息
- 与消息存储解耦

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                     Session Service                          │
│                                                              │
│  ┌──────────────────┐              ┌──────────────────┐    │
│  │  Message Repo    │              │ Context Manager  │    │
│  │  (Complete       │              │ (LLM Context     │    │
│  │   History)       │              │  Window)         │    │
│  └──────────────────┘              └──────────────────┘    │
│         │                                    │              │
│         │ Save all messages                  │ Manage LLM  │
│         │ for display                        │ context     │
│         ▼                                    ▼              │
│  ┌──────────────────┐              ┌──────────────────┐    │
│  │   Database       │              │  In-Memory       │    │
│  │   (Persistent)   │              │  (Session-based) │    │
│  └──────────────────┘              └──────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## 压缩策略

### 1. 滑动窗口策略 (Sliding Window)

保留最近的 N 条消息，丢弃更早的消息。

**优点:**
- 简单高效
- 不需要额外的 LLM 调用
- 适合短期对话

**配置示例:**
```json
{
  "enabled": true,
  "max_tokens": 8192,
  "compression_strategy": "sliding_window",
  "recent_message_count": 20
}
```

**工作原理:**
```
原始消息: [system, msg1, msg2, msg3, ..., msg18, msg19, msg20, msg21, msg22]
                                              ↑
                                     保留最近20条消息
压缩结果: [system, msg3, msg4, ..., msg20, msg21, msg22]
          └─────┘  └──────────────────────────────────┘
         保留系统消息      保留最近的20条消息
```

### 2. 智能压缩策略 (Smart Compression)

使用 LLM 总结旧消息，保留最近消息的完整内容。

**优点:**
- 保留历史关键信息
- 更好的上下文连贯性
- 适合长期对话

**配置示例:**
```json
{
  "enabled": true,
  "max_tokens": 8192,
  "compression_strategy": "smart",
  "recent_message_count": 10
}
```

**工作原理:**
```
原始消息: [system, msg1, msg2, ..., msg15, msg16, msg17, msg18, msg19, msg20]
                  └───────────────┘          └─────────────────────────────┘
                    旧消息（总结）                    最近消息（保留）

压缩结果: [system, summary, msg16, msg17, msg18, msg19, msg20]
          └─────┘  └──────┘ └──────────────────────────────┘
         系统消息    总结     保留最近的10条消息（完整）
```

## 使用方法

### 1. 默认配置

如果不设置 `context_config`，系统使用默认配置：
- 最大 Token: 8192
- 策略: 滑动窗口
- 保留消息数: 20

```go
// 不需要特别配置，自动使用默认设置
session := &types.Session{
    TenantID:        tenantID,
    KnowledgeBaseID: kbID,
    // context_config 为 nil，使用默认配置
}
```

### 2. 自定义配置 - 滑动窗口

```go
session := &types.Session{
    TenantID:        tenantID,
    KnowledgeBaseID: kbID,
    ContextConfig: &types.ContextConfig{
        Enabled:             true,
        MaxTokens:           16384,  // GPT-4 的上下文窗口
        CompressionStrategy: types.ContextCompressionSlidingWindow,
        RecentMessageCount:  30,     // 保留最近30条消息
    },
}
```

### 3. 自定义配置 - 智能压缩

```go
session := &types.Session{
    TenantID:        tenantID,
    KnowledgeBaseID: kbID,
    SummaryModelID:  "gpt-4",  // 用于总结的模型
    ContextConfig: &types.ContextConfig{
        Enabled:             true,
        MaxTokens:           8192,
        CompressionStrategy: types.ContextCompressionSmart,
        RecentMessageCount:  15,  // 保留最近15条完整消息
    },
}
```

### 4. 通过 API 创建/更新会话

```bash
# 创建带上下文配置的会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 1" \
  -d '{
    "knowledge_base_id": "kb-123",
    "context_config": {
      "enabled": true,
      "max_tokens": 8192,
      "compression_strategy": "sliding_window",
      "recent_message_count": 20
    }
  }'
```

## 工作流程

### 对话流程

```
1. 用户发送消息
   ↓
2. 保存消息到数据库 (Message Repo)
   ↓
3. 添加消息到上下文管理器 (Context Manager)
   ↓
4. Context Manager 检查 Token 限制
   ↓
5. 如果超出限制，应用压缩策略
   ↓
6. 获取压缩后的上下文
   ↓
7. 发送给 LLM
   ↓
8. 保存响应到数据库
   ↓
9. 添加响应到上下文管理器
```

### 代码示例

```go
// 在 AgentQA 中的使用
func (s *sessionService) AgentQA(ctx context.Context, sessionID, query string, assistantMessageID string) (
    []*types.SearchResult, <-chan types.StreamResponse, error,
) {
    // 1. 获取会话配置
    session, err := s.sessionRepo.Get(ctx, tenantID, sessionID)
    
    // 2. 从上下文管理器获取 LLM 上下文（自动应用压缩）
    history, err := s.getContextForSession(ctx, session, sessionID)
    // history 是压缩后的，适合发送给 LLM 的消息列表
    
    // 3. 执行 Agent
    eventChan, err := engine.ExecuteStreamWithHistory(ctx, query, history)
    
    // 4. 保存新消息后，添加到上下文管理器
    // s.AddMessageToContext(ctx, session, sessionID, newMessage)
    
    return searchResults, responseChan, nil
}
```

## 配置参数说明

### ContextConfig 字段

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `enabled` | bool | 是否启用上下文管理 | true |
| `max_tokens` | int | 最大 Token 数 | 8192 |
| `compression_strategy` | string | 压缩策略: "sliding_window" 或 "smart" | "sliding_window" |
| `recent_message_count` | int | 保留的最近消息数 | 20 (sliding_window) <br> 10 (smart) |

### 推荐配置

#### 短期对话（客服、问答）
```json
{
  "enabled": true,
  "max_tokens": 4096,
  "compression_strategy": "sliding_window",
  "recent_message_count": 15
}
```

#### 长期对话（咨询、助手）
```json
{
  "enabled": true,
  "max_tokens": 8192,
  "compression_strategy": "smart",
  "recent_message_count": 10
}
```

#### 高端模型（GPT-4 Turbo, Claude）
```json
{
  "enabled": true,
  "max_tokens": 16384,
  "compression_strategy": "smart",
  "recent_message_count": 20
}
```

## 监控和调试

### 查看上下文统计

```go
// 获取上下文统计信息
stats, err := contextManager.GetContextStats(ctx, sessionID)
if err == nil {
    log.Printf("Session %s context stats:", sessionID)
    log.Printf("  Messages: %d", stats.MessageCount)
    log.Printf("  Tokens: ~%d", stats.TokenCount)
    log.Printf("  Compressed: %v", stats.IsCompressed)
    log.Printf("  Original messages: %d", stats.OriginalMessageCount)
}
```

### 日志示例

```
INFO  Using custom context config for session abc123: strategy=smart, max_tokens=8192, recent_count=10
INFO  Context exceeds max tokens (9500 > 8192), applying compression
INFO  Summarizing 15 old messages
INFO  Successfully summarized 15 messages
INFO  Smart compression: 25 -> 11 messages (system: 1, compressed: 1, recent: 10)
INFO  LLM context stats for session abc123: messages=11, tokens=~7800, compressed=true
```

## 与消息存储的区别

| 特性 | 消息存储 (Message Repo) | 上下文管理器 (Context Manager) |
|------|------------------------|------------------------------|
| 目的 | 完整保存对话历史 | 管理 LLM 输入上下文 |
| 存储 | 数据库（持久化） | 内存（会话级别） |
| 内容 | 所有消息（完整） | 压缩后的消息 |
| 大小限制 | 无限制 | 受 Token 限制 |
| 使用场景 | 展示历史、审计 | LLM 推理输入 |
| 生命周期 | 永久保存 | 会话期间 |

## 最佳实践

1. **选择合适的策略**:
   - 短对话 → 滑动窗口（性能更好）
   - 长对话 → 智能压缩（保留更多上下文）

2. **配置 Token 限制**:
   - 设置为模型上下文窗口的 70-80%
   - 为响应留出足够空间

3. **调整保留消息数**:
   - 滑动窗口: 15-30 条消息
   - 智能压缩: 8-15 条最近消息

4. **监控压缩效果**:
   - 定期检查 Token 使用情况
   - 观察压缩是否影响对话质量

5. **性能考虑**:
   - 智能压缩会额外调用 LLM（有成本）
   - 滑动窗口无额外开销

## 数据库迁移

运行以下迁移脚本添加 `context_config` 字段：

### MySQL
```bash
mysql -u root -p your_database < migrations/mysql/06-add-context-config-to-sessions.sql
```

### ParadeDB/PostgreSQL
```bash
psql -U postgres -d your_database -f migrations/paradedb/06-add-context-config-to-sessions.sql
```

## 常见问题

### Q1: 上下文管理器的数据会持久化吗？
**A**: 不会。上下文管理器是内存级别的，按 Session 维护。重启后会清空，需要从消息历史重新构建。

### Q2: 如何清空某个会话的上下文？
**A**: 
```go
err := contextManager.ClearContext(ctx, sessionID)
```

### Q3: 压缩后的消息会影响数据库中的消息吗？
**A**: 不会。压缩只影响发送给 LLM 的上下文，数据库中的消息完整保留。

### Q4: 智能压缩的总结质量如何保证？
**A**: 
- 使用会话配置的 `summary_model_id` 模型
- 可以使用更强的模型（如 GPT-4）进行总结
- 总结时使用低 temperature (0.3) 确保一致性

### Q5: 如何为现有会话启用上下文管理？
**A**: 
```bash
# 更新会话配置
curl -X PUT http://localhost:8080/api/v1/sessions/{session_id} \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 1" \
  -d '{
    "context_config": {
      "enabled": true,
      "max_tokens": 8192,
      "compression_strategy": "sliding_window",
      "recent_message_count": 20
    }
  }'
```

## 未来改进

- [ ] 支持更多压缩策略（如 MapReduce 总结）
- [ ] 支持自定义 Token 计算器
- [ ] 持久化压缩后的上下文摘要
- [ ] 支持跨会话的上下文共享
- [ ] 添加更详细的压缩指标和监控
- [ ] 支持上下文恢复机制

## 参考

- [Agent 文档](./AGENT.md)
- [API 文档](./API.md)
- [消息管理](../internal/types/message.go)

