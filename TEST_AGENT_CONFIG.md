# Agent 配置功能测试指南

## 快速测试步骤

### 准备工作

1. **确保数据库已迁移**

   ```bash
   # MySQL
   mysql -h localhost -u root -p weknora < migrations/mysql/02-add-agent-support.sql
   
   # 或 PostgreSQL
   psql -h localhost -U postgres -d weknora -f migrations/paradedb/02-add-agent-support.sql
   ```

2. **启动后端服务**

   ```bash
   cd /Users/wizard/code/go/src/git.woa.com/wxg-prc/WeKnora
   go run cmd/server/main.go
   ```

3. **启动前端服务**

   ```bash
   cd /Users/wizard/code/go/src/git.woa.com/wxg-prc/WeKnora/frontend
   npm run dev
   ```

### 测试场景

#### 场景 1：Agent 配置保存到后台

**步骤：**

1. 打开浏览器访问：`http://localhost:5173`（或前端服务地址）
2. 导航到 **设置** 页面
3. 在 **Agent 智能问答模式** 区域：
   - 打开 Agent 模式开关
   - 设置最大推理步数为 `8`
   - 调整温度参数为 `0.8`
   - 选择允许的工具
4. 点击 **保存配置** 按钮
5. 验证提示"配置保存成功"

**验证：**

- 打开浏览器开发者工具（F12）
- 查看 Network 标签，找到 `PUT /api/v1/tenants/1/agent-config` 请求
- 验证请求状态为 200
- 查看响应内容确认配置已保存

**数据库验证：**

```sql
SELECT agent_config FROM tenants WHERE id = 1;
```

预期输出类似：
```json
{
  "enabled": true,
  "max_iterations": 8,
  "temperature": 0.8,
  "allowed_tools": ["knowledge_search", "multi_kb_search", "list_knowledge_bases"],
  ...
}
```

---

#### 场景 2：刷新页面后配置保持

**步骤：**

1. 完成场景 1 的配置保存
2. 刷新浏览器页面（F5）
3. 再次进入 **设置** 页面

**验证：**

- Agent 模式开关保持启用状态
- 最大推理步数显示为 `8`
- 温度参数显示为 `0.8`
- 工具选择保持不变

---

#### 场景 3：快速切换按钮功能

**步骤：**

1. 导航到对话页面（选择任意知识库和会话）
2. 观察输入框左下角的机器人图标 🤖
3. 点击机器人图标

**验证：**

- **初始状态（禁用）：**
  - 图标灰色，半透明
  - 悬停时背景变浅蓝色

- **点击后（启用）：**
  - 图标变为彩色
  - 有呼吸动画效果
  - 显示提示"Agent 智能模式已启用"
  - 页面顶部出现 "Agent 智能模式" 指示器（绿色标签）

- **再次点击（禁用）：**
  - 图标变回灰色
  - 显示提示"Agent 智能模式已禁用"
  - 顶部指示器消失

---

#### 场景 4：Agent 模式对话测试

**步骤：**

1. 在对话页面，确保 Agent 模式已启用（机器人图标彩色）
2. 输入一个复杂问题，例如：
   ```
   请对比文档 A 和文档 B 的核心观点，并给出详细分析
   ```
3. 发送消息

**验证：**

- 能看到 Agent 的思考过程（thinking steps）
- 显示工具调用过程（tool calls）
- 最终给出综合答案
- 请求 URL 为 `/api/v1/agent-chat/:session_id`（而非 `/api/v1/knowledge-chat`）

**对比测试：**

1. 禁用 Agent 模式（点击机器人图标）
2. 发送相同的问题
3. 验证使用普通 QA 模式（直接返回答案，无思考步骤）
4. 请求 URL 为 `/api/v1/knowledge-chat/:session_id`

---

#### 场景 5：UI 样式验证

**设置页面样式：**

- ✅ 页面标题左侧有绿色竖线装饰
- ✅ 配置区域使用浅绿色渐变背景
- ✅ Agent 配置区域有绿色左边框
- ✅ 主按钮使用绿色渐变 (#07c05f → #00a651)
- ✅ 按钮悬停时有上浮动画效果
- ✅ 表单输入框聚焦时有绿色边框和阴影

**输入框样式：**

- ✅ 机器人图标位于左下角
- ✅ 禁用时灰色半透明
- ✅ 启用时彩色有动画
- ✅ 悬停时有缩放效果

---

## 常见问题排查

### 问题 1：保存配置时出现 404 错误

**原因：** 后端路由未正确注册

**排查：**
```bash
# 检查后端日志，确认路由是否注册
curl -X GET http://localhost:8080/api/v1/tenants/1/agent-config
```

**解决：** 重新编译并启动后端服务

---

### 问题 2：刷新后配置丢失

**原因：** 可能只保存到 localStorage，未保存到后端

**排查：**
```sql
-- 查询数据库
SELECT agent_config FROM tenants WHERE id = 1;
```

**解决：** 确保 `onSubmit` 方法中调用了 `updateAgentConfig` API

---

### 问题 3：快速切换按钮不显示

**原因：** 前端代码未更新或缓存问题

**排查：**
```bash
# 检查 Input-field.vue 是否包含切换按钮代码
grep -n "answers-input-agent" frontend/src/components/Input-field.vue
```

**解决：**
1. 清除浏览器缓存
2. 强制刷新（Ctrl+Shift+R）
3. 重新构建前端

---

### 问题 4：Agent 模式不生效

**原因：** Session 创建时未传递 Agent 配置

**排查：**
```javascript
// 浏览器控制台
console.log(useSettingsStore().isAgentEnabled)
```

**解决：** 确保 `chat/index.vue` 中正确读取 `isAgentEnabled` 状态

---

## 性能测试

### 配置保存性能

```bash
# 测试 API 响应时间
time curl -X PUT http://localhost:8080/api/v1/tenants/1/agent-config \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "max_iterations": 5,
    "temperature": 0.7,
    "allowed_tools": ["knowledge_search"]
  }'
```

预期响应时间：< 100ms

---

### 配置读取性能

```bash
# 测试 API 响应时间
time curl -X GET http://localhost:8080/api/v1/tenants/1/agent-config
```

预期响应时间：< 50ms

---

## 浏览器兼容性测试

- ✅ Chrome/Edge (最新版)
- ✅ Firefox (最新版)
- ✅ Safari (最新版)

---

## 测试完成清单

- [ ] 场景 1：Agent 配置保存到后台
- [ ] 场景 2：刷新页面后配置保持
- [ ] 场景 3：快速切换按钮功能
- [ ] 场景 4：Agent 模式对话测试
- [ ] 场景 5：UI 样式验证
- [ ] 数据库迁移成功
- [ ] 后端 API 正常响应
- [ ] 前端页面正常显示
- [ ] 浏览器控制台无错误
- [ ] 所有配置正确保存和加载

---

## 测试报告模板

```markdown
## Agent 配置功能测试报告

**测试日期：** YYYY-MM-DD
**测试人员：** [姓名]
**环境：** [开发/测试/生产]

### 测试结果

| 场景 | 状态 | 备注 |
|------|------|------|
| Agent 配置保存 | ✅ / ❌ | |
| 配置持久化 | ✅ / ❌ | |
| 快速切换按钮 | ✅ / ❌ | |
| Agent 模式对话 | ✅ / ❌ | |
| UI 样式 | ✅ / ❌ | |

### 发现的问题

1. [问题描述]
   - **严重程度：** 高/中/低
   - **重现步骤：** ...
   - **预期结果：** ...
   - **实际结果：** ...

### 建议改进

1. [改进建议]

### 总体评价

[整体功能评价]
```

---

## 下一步

完成测试后，如果一切正常，可以：

1. 提交代码到版本控制
2. 创建 Pull Request
3. 通知团队进行 Code Review
4. 部署到测试环境
5. 准备发布说明

---

**注意事项：**
- 测试过程中请记录所有错误和警告信息
- 截图保存关键步骤和结果
- 如有问题，请查看 AGENT_CONFIG_DEPLOYMENT.md 中的常见问题部分

