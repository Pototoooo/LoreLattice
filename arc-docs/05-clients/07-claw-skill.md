# Claw Skill

Claw Skill 是把 LoreLattice 挂给 AI Agent 用的一种方式：安装之后，OpenClaw 生态里的 Agent 就能通过 LoreLattice 的 REST API 往知识库里写内容、跨库检索。

本文不推荐未经验证的外部 Skill 包。接入时请使用与当前实例 API 兼容、来源明确的工具；需要可核对源码的接入方式，可参阅 [MCP 集成](../03-features/08-mcp.md)。

## 能做什么

| 能力 | 对应接口 |
| --- | --- |
| 上传文件 | 把 PDF / Word / Excel 等文档送进知识库并自动解析向量化 |
| 导入网页 | 按 URL 抓取正文写入知识库，支持轮询解析状态 |
| 写入 Markdown | 以 Markdown 创建或编辑知识条目，适合会议记录、结构化笔记 |
| 混合检索 | 单库 `hybrid-search` 与跨库 `knowledge-search`，向量 + 关键词召回 |
| 浏览知识库 | 列出知识库与条目、查看详情 |

## 怎么配

LoreLattice 界面里有引导页：「设置 → 集成 → Claw Skill」，会带上当前实例的 API 地址与可复制的环境变量示例、安装命令。步骤：

1. **拿 API 凭证**：「设置 → API 信息」里复制 API Key 与 API 地址；
2. **配环境变量**：在终端或 `~/.zshrc` / `~/.bashrc` 里设置

   ```bash
   export LORELATTICE_BASE_URL=https://your-lorelattice.example.com/api/v1
   export LORELATTICE_API_KEY=sk-xxxxx
   ```

3. **选择接入实现**：先核对工具与当前 API 的兼容性；本文不提供外部 Skill 包的安装命令；
4. **验证**：让 Agent 列一次知识库或跑一次检索，确认凭证与网络可达。

## 和 MCP 的关系

两者都是「把 LoreLattice 给外部 Agent 用」，选哪个取决于对方生态：

| | Claw Skill | MCP Server |
| --- | --- | --- |
| 面向 | OpenClaw / ClawHub 生态的 Agent | 支持 MCP 协议的客户端（Claude Desktop、VS Code Copilot 等） |
| 安装 | 由所选兼容实现提供，本文不指定外部包 | 从本仓库 `mcp-server/` 执行 `pip install .` |
| 传输 | 直接调 REST | stdio / SSE / Streamable HTTP |
| 能力范围 | 导入、检索、浏览（5 类） | 29 个工具，另含租户、模型、会话、Agent 问答、Wiki |
| 文档 | 本篇 | [MCP 集成](../03-features/08-mcp.md) |

需要更完整的能力（跑 Agent 对话、管模型、读 Wiki）时用 MCP Server；只是想让 Agent 存取资料，Skill 更轻。

## 相关

- 凭证与能力收窄：[租户、用户与认证授权](../03-features/01-tenant-auth.md)
- 底层接口：[API 总览](../04-api/01-api-overview.md)
- 其它集成方式：[Chrome 插件](06-chrome-extension.md)、[MCP 集成](../03-features/08-mcp.md)
