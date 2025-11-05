# Stream Manager 重构：从内容累积到事件流

## 问题分析

用户指出了一个关键问题：当前只有 `handleFinalAnswer` 在更新 stream，但其他事件（thinking、tool_call、tool_result 等）也应该推送到 stream。这样当用户刷新页面时，可以**恢复完整的输出过程**。

### 之前的问题 ❌

```go
// StreamInfo 只存储累积的内容
type StreamInfo struct {
    SessionID           string
    RequestID           string
    Query               string
    Content             string           // ❌ 只有累积内容，丢失了事件流
    KnowledgeReferences types.References
    LastUpdated         time.Time
    IsCompleted         bool
}

// UpdateStream 只能追加内容
func UpdateStream(ctx, sessionID, requestID string, 
    content string, references types.References) error
```

**问题：**
1. ❌ 只保存最终内容，没有保存事件流
2. ❌ 刷新页面后，只能看到累积的答案，看不到思考/工具调用过程
3. ❌ 无法重现完整的 Agent 执行过程
4. ❌ 用户体验差：刷新后丢失所有中间状态

## 改进方案 ✅

### 核心思想

**从存储内容到存储事件流** - 每个事件（thinking、tool_call、tool_result 等）都推送到stream，刷新时可以完整重放。

### 新的数据结构

```go
// StreamEvent 代表一个事件
type StreamEvent struct {
    Type      types.ResponseType     // 事件类型
    Content   string                 // 事件内容
    Done      bool                   // 是否完成
    Timestamp time.Time              // 事件时间
    Data      map[string]interface{} // 附加数据
}

// StreamInfo 存储完整的事件流
type StreamInfo struct {
    SessionID           string
    RequestID           string
    Query               string
    Events              []StreamEvent      // ✅ 完整的事件流
    KnowledgeReferences types.References
    LastUpdated         time.Time
    IsCompleted         bool
}
```

### 新的接口

```go
type StreamManager interface {
    // RegisterStream 注册新流
    RegisterStream(ctx, sessionID, requestID, query string) error

    // PushEvent 推送单个事件 ✅
    PushEvent(ctx, sessionID, requestID string, event StreamEvent) error

    // UpdateReferences 更新知识引用 ✅
    UpdateReferences(ctx, sessionID, requestID string, references types.References) error

    // CompleteStream 完成流
    CompleteStream(ctx, sessionID, requestID string) error

    // GetStream 获取流（用于刷新重放）✅
    GetStream(ctx, sessionID, requestID string) (*StreamInfo, error)
}
```

## 实现详情

### 1. AgentStreamHandler - 推送所有事件

#### handleThought

```go
func (h *AgentStreamHandler) handleThought(...) error {
    // 1. 发送 SSE
    h.ginContext.SSEvent("message", response)
    h.ginContext.Writer.Flush()

    // 2. 推送事件到 stream（用于刷新重放）✅
    h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
        Type:      types.ResponseTypeThinking,
        Content:   data.Content,
        Done:      data.Done,
        Timestamp: time.Now(),
    })
}
```

#### handleToolCall

```go
func (h *AgentStreamHandler) handleToolCall(...) error {
    // 1. 发送 SSE
    h.ginContext.SSEvent("message", response)
    
    // 2. 推送事件到 stream（含工具调用详情）✅
    h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
        Type:      types.ResponseTypeToolCall,
        Content:   fmt.Sprintf("Calling tool: %s", data.ToolName),
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "tool_name": data.ToolName,
            "arguments": data.Arguments,
        },
    })
}
```

#### handleToolResult

```go
func (h *AgentStreamHandler) handleToolResult(...) error {
    // 1. 发送 SSE
    h.ginContext.SSEvent("message", response)
    
    // 2. 推送事件到 stream（含结果详情）✅
    h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
        Type:      responseType,
        Content:   content,
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "tool_name": data.ToolName,
            "success":   data.Success,
            "output":    data.Output,
            "error":     data.Error,
        },
    })
}
```

#### handleReflection

```go
func (h *AgentStreamHandler) handleReflection(...) error {
    // 1. 发送 SSE
    h.ginContext.SSEvent("message", response)
    
    // 2. 推送事件到 stream ✅
    h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
        Type:      "reflection",
        Content:   data.Content,
        Timestamp: time.Now(),
    })
}
```

#### handleFinalAnswer

```go
func (h *AgentStreamHandler) handleFinalAnswer(...) error {
    // 1. 发送 SSE
    h.ginContext.SSEvent("message", response)
    
    // 2. 推送事件到 stream ✅
    h.streamManager.PushEvent(h.ctx, h.sessionID, h.assistantMessageID, interfaces.StreamEvent{
        Type:      types.ResponseTypeAnswer,
        Content:   data.Content,
        Done:      data.Done,
        Timestamp: time.Now(),
    })
}
```

### 2. 刷新重放逻辑

#### 之前 ❌ - 只显示累积内容

```go
// 只发送累积的内容
if streamInfo.Content != "" {
    c.SSEvent("message", &types.StreamResponse{
        Content: streamInfo.Content,  // ❌ 失去中间过程
    })
}
```

#### 现在 ✅ - 完整重放事件流

```go
// 重放所有事件
if len(streamInfo.Events) > 0 {
    logger.Debugf(ctx, "Replaying %d existing events", len(streamInfo.Events))
    for _, evt := range streamInfo.Events {
        c.SSEvent("message", &types.StreamResponse{
            ID:           message.RequestID,
            ResponseType: evt.Type,
            Content:      evt.Content,
            Done:         evt.Done,
        })
    }
}
```

### 3. 实时监控新事件

#### 之前 ❌ - 监控内容增量

```go
// 计算内容差异
if len(latestStreamInfo.Content) > len(currentContent) {
    newContent := latestStreamInfo.Content[len(currentContent):]
    contentCh <- newContent
    currentContent = latestStreamInfo.Content
}
```

#### 现在 ✅ - 监控新事件

```go
// 检测新事件
if len(latestStreamInfo.Events) > currentEventCount {
    eventIndexCh <- currentEventCount
    currentEventCount = len(latestStreamInfo.Events)
}

// 发送新事件
case startIdx := <-eventIndexCh:
    for i := startIdx; i < len(latestStreamInfo.Events); i++ {
        evt := latestStreamInfo.Events[i]
        c.SSEvent("message", &types.StreamResponse{
            ResponseType: evt.Type,
            Content:      evt.Content,
            Done:         evt.Done,
        })
    }
```

## 数据流对比

### 之前 ❌

```
Agent 执行
  │
  ├─→ thinking event → SSE → 前端 (丢失)
  ├─→ tool_call event → SSE → 前端 (丢失)
  ├─→ tool_result event → SSE → 前端 (丢失)
  └─→ final_answer event → SSE → 前端
                          └─→ UpdateStream(累积内容)

用户刷新 → GetStream → 只有累积内容 ❌
```

### 现在 ✅

```
Agent 执行
  │
  ├─→ thinking event → SSE → 前端
  │                  └─→ PushEvent(thinking) ✅
  ├─→ tool_call event → SSE → 前端
  │                   └─→ PushEvent(tool_call) ✅
  ├─→ tool_result event → SSE → 前端
  │                     └─→ PushEvent(tool_result) ✅
  └─→ final_answer event → SSE → 前端
                          └─→ PushEvent(final_answer) ✅

用户刷新 → GetStream → 完整事件流 ✅
         → 重放所有事件
```

## 优势总结

| 方面 | 之前 ❌ | 现在 ✅ |
|------|---------|---------|
| **事件保留** | 只保留最终内容 | 保留完整事件流 |
| **刷新体验** | 丢失中间过程 | 完整重放 |
| **思考过程** | 不可见 | 可见 |
| **工具调用** | 不可见 | 可见 |
| **错误调试** | 困难 | 容易（完整历史） |
| **用户体验** | 差 | 好 |

## 存储实现

### Memory Manager

```go
type memoryStreamInfo struct {
    sessionID           string
    requestID           string
    query               string
    events              []interfaces.StreamEvent  // ✅ 事件数组
    knowledgeReferences types.References
    lastUpdated         time.Time
    isCompleted         bool
}

func (m *MemoryStreamManager) PushEvent(..., event interfaces.StreamEvent) error {
    stream.events = append(stream.events, event)  // ✅ 追加事件
    stream.lastUpdated = time.Now()
    return nil
}
```

### Redis Manager

```go
type redisStreamInfo struct {
    SessionID           string                      `json:"session_id"`
    RequestID           string                      `json:"request_id"`
    Query               string                      `json:"query"`
    Events              []interfaces.StreamEvent    `json:"events"`  // ✅ 事件数组
    KnowledgeReferences types.References            `json:"knowledge_references"`
    LastUpdated         time.Time                   `json:"last_updated"`
    IsCompleted         bool                        `json:"is_completed"`
}

func (r *RedisStreamManager) PushEvent(..., event interfaces.StreamEvent) error {
    info.Events = append(info.Events, event)  // ✅ 追加事件
    updatedData, _ := json.Marshal(info)
    return r.client.Set(ctx, key, updatedData, r.ttl).Err()
}
```

## 使用场景

### 场景 1: 正常执行

1. Agent 执行过程中，每个事件同时：
   - 发送 SSE 给前端（实时显示）
   - PushEvent 到 stream（持久化）

### 场景 2: 刷新页面

1. 前端重新连接
2. GetStream 获取完整事件流
3. 重放所有事件
4. 继续监听新事件

### 场景 3: 断线重连

1. 客户端记录已接收的事件数量
2. 重连后从 `Events[lastIndex:]` 开始重放
3. 无缝恢复

## 前端适配

```typescript
// 刷新时恢复状态
const streamInfo = await api.getStream(sessionId, messageId);

// 重放所有历史事件
streamInfo.events.forEach(event => {
    switch (event.type) {
        case 'thinking':
            showThinking(event.content);
            break;
        case 'tool_call':
            showToolCall(event.data.tool_name, event.data.arguments);
            break;
        case 'tool_result':
            showToolResult(event.data.tool_name, event.data.output);
            break;
        case 'answer':
            showAnswer(event.content);
            break;
    }
});

// 继续监听新事件
connectSSE(sessionId, messageId);
```

## 性能考虑

### 内存占用

- 每个事件 ~100-500 bytes
- 典型的 Agent 执行 ~20-50 个事件
- 总计 ~2-25 KB per session
- 可接受

### Redis 存储

- JSON 序列化事件数组
- TTL 控制：默认 1 小时后自动清理
- 压缩：Redis 自动处理

### 优化策略

1. **事件限制**: 可选地限制最大事件数（如 1000个）
2. **压缩**: 对大量事件进行压缩
3. **分片**: 超大事件流可以分片存储

---

**重构完成时间**: 2025-10-31  
**状态**: ✅ 完成，支持完整事件流重放

